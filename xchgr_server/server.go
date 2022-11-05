package xchgr_server

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

type Router struct {
	// Sync
	mtx sync.Mutex

	// State
	started  bool
	stopping bool

	// Data
	nonces *Nonces

	network *Network
	nextId  uint64

	addresses map[string]*AddressStorage

	// Statistics
	stat       RouterStatistics
	statLast   RouterStatistics
	statLastDT time.Time
	statSpeed  RouterSpeedStatistics

	lastDebugInfo []byte

	clearAddressesLastDT time.Time
}

type RouterStatistics struct {
	FramesIn  int `json:"frames_in"`
	FramesOut int `json:"frames_out"`
	BytesIn   int `json:"bytes_in"`
	BytesOut  int `json:"bytes_out"`
}

type RouterSpeedStatistics struct {
	SpeedFramesIn  int `json:"frames_in"`
	SpeedFramesOut int `json:"frames_out"`
	SpeedBytesIn   int `json:"bytes_in"`
	SpeedBytesOut  int `json:"bytes_out"`

	SpeedBytesKIn  int `json:"kilobytes_in"`
	SpeedBytesKOut int `json:"kilobytes_out"`

	SpeedBytesMIn  int `json:"megabytes_in"`
	SpeedBytesMOut int `json:"megabytes_out"`
}

const (
	NONCE_COUNT       = 1024 * 1024
	INPUT_BUFFER_SIZE = 1024 * 1024
	STORING_TIMEOUT   = 60 * time.Second
)

func NewRouter() *Router {
	var c Router
	c.network = NewNetworkDefault()
	c.nonces = NewNonces(100000)
	c.addresses = make(map[string]*AddressStorage)

	c.statLastDT = time.Now()
	c.clearAddressesLastDT = time.Now()
	return &c
}

func (c *Router) Start() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// Checks
	if c.started {
		return errors.New("already started")
	}
	if c.stopping {
		return errors.New("it is stopping")
	}

	go c.thBackgroundOperations()

	return nil
}

func (c *Router) Stop() error {
	c.mtx.Lock()
	if !c.started {
		c.mtx.Unlock()
		return errors.New("already stopped")
	}
	if c.stopping {
		c.mtx.Unlock()
		return errors.New("already stopping")
	}
	c.stopping = true
	c.mtx.Unlock()

	for {
		c.mtx.Lock()
		started := c.started
		c.mtx.Unlock()
		if !started {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	c.mtx.Lock()
	c.nonces = nil
	c.mtx.Unlock()

	return nil
}

func (c *Router) NetworkBS() (result []byte, err error) {
	c.mtx.Lock()
	result, err = json.MarshalIndent(c.network, "", " ")
	c.mtx.Unlock()
	return
}

func (c *Router) thBackgroundOperations() {
	for {
		c.mtx.Lock()
		stopping := c.stopping
		c.mtx.Unlock()
		if stopping {
			break
		}
		time.Sleep(50 * time.Millisecond)
		c.thStatistics()
		c.thClearAddresses()
	}
}

func (c *Router) thStatistics() {
	now := time.Now()
	if now.Sub(c.statLastDT) >= 1*time.Second {
		c.mtx.Lock()
		var stat RouterStatistics
		stat.BytesIn = c.stat.BytesIn - c.statLast.BytesIn
		stat.BytesOut = c.stat.BytesOut - c.statLast.BytesOut
		stat.FramesIn = c.stat.FramesIn - c.statLast.FramesIn
		stat.FramesOut = c.stat.FramesOut - c.statLast.FramesOut
		c.statLast = c.stat
		c.mtx.Unlock()

		c.statSpeed.SpeedBytesIn = int(float64(stat.BytesIn) / now.Sub(c.statLastDT).Seconds())
		c.statSpeed.SpeedBytesOut = int(float64(stat.BytesOut) / now.Sub(c.statLastDT).Seconds())
		c.statSpeed.SpeedFramesIn = int(float64(stat.FramesIn) / now.Sub(c.statLastDT).Seconds())
		c.statSpeed.SpeedFramesOut = int(float64(stat.FramesOut) / now.Sub(c.statLastDT).Seconds())
		c.statSpeed.SpeedBytesKIn = c.statSpeed.SpeedBytesIn / 1024
		c.statSpeed.SpeedBytesKOut = c.statSpeed.SpeedBytesOut / 1024
		c.statSpeed.SpeedBytesMIn = c.statSpeed.SpeedBytesIn / (1024 * 1024)
		c.statSpeed.SpeedBytesMOut = c.statSpeed.SpeedBytesOut / (1024 * 1024)
		c.statLastDT = now
	}

	c.buildDebugString()
}

func (c *Router) thClearAddresses() {
	now := time.Now()
	if now.Sub(c.clearAddressesLastDT) >= 1*time.Second {
		c.mtx.Lock()
		addresses := make([]*AddressStorage, 0)
		for address, addressStorage := range c.addresses {
			if now.Sub(addressStorage.TouchDT) > 10*time.Second {
				delete(c.addresses, address)
				continue
			}
			addresses = append(addresses, addressStorage)
		}
		c.mtx.Unlock()

		for _, a := range addresses {
			a.Clear()
		}

		c.clearAddressesLastDT = now
	}
}

func (c *Router) Put(frame []byte) {
	var ok bool
	var addressStorage *AddressStorage

	addressDestBS := frame[70:100]
	addressDest := "#" + strings.ToLower(base32.StdEncoding.EncodeToString(addressDestBS))
	//fmt.Println("FRAME to ", addressDest, frame[8])

	c.mtx.Lock()
	addressStorage, ok = c.addresses[addressDest]
	if !ok || addressStorage == nil {
		addressStorage = NewAddressStorage()
		c.addresses[addressDest] = addressStorage
	}
	id := c.nextId
	c.nextId++
	c.mtx.Unlock()

	addressStorage.Put(id, frame)
	c.stat.FramesIn++
	c.stat.BytesIn += len(frame)
}

// Get message request
func (c *Router) GetMessages(frame []byte) (response []byte, err error) {
	var ok bool
	var addressStorage *AddressStorage

	if len(frame) < 46 {
		err = errors.New("wrong frame size")
		return
	}

	afterId := binary.LittleEndian.Uint64(frame[0:])
	maxSize := binary.LittleEndian.Uint64(frame[8:])
	addressSrcBS := frame[16 : 16+30]

	addressSrc := "#" + strings.ToLower(base32.StdEncoding.EncodeToString(addressSrcBS))

	c.mtx.Lock()
	addressStorage, ok = c.addresses[addressSrc]
	c.mtx.Unlock()

	if !ok || addressStorage == nil {
		response = make([]byte, 8)
		binary.LittleEndian.PutUint64(response[0:], 0)
		return
	}

	msgData, lastId, count := addressStorage.GetMessage(afterId, maxSize)
	response = make([]byte, 8+len(msgData))
	binary.LittleEndian.PutUint64(response[0:], lastId)
	if msgData != nil {
		copy(response[8:], msgData)
	}

	c.stat.FramesOut += count
	c.stat.BytesOut += len(msgData)
	return
}

func RSAPublicKeyFromDer(publicKeyDer []byte) (publicKey *rsa.PublicKey, err error) {
	publicKey, err = x509.ParsePKCS1PublicKey(publicKeyDer)
	return
}

func (c *Router) resolveAddress(address string) (nativeAddress string, err error) {
	if len(address) < 1 {
		err = errors.New("empty address")
		return
	}

	if address[0] == '#' {
		nativeAddress = address
		return
	}

	if strings.HasSuffix(address, ".xchg") {
		if address == "42.xchg" {
			nativeAddress = "kqfc2fwogggtlsf7vnh46hhgdjmheiqvqycapj2f2xe2d5jz"
			return
		}
	}

	err = errors.New("unknown address")
	return
}

func (c *Router) DebugString() (result []byte) {
	c.mtx.Lock()
	result = make([]byte, len(c.lastDebugInfo))
	copy(result, c.lastDebugInfo)
	c.mtx.Unlock()
	return
}

func (c *Router) buildDebugString() {
	type AddressInfo struct {
		Address      string `json:"address"`
		MessageCount int    `json:"messages"`
	}

	type DebugInfo struct {
		AddressCount int                   `json:"address_count"`
		NextMsgId    int                   `json:"next_msg_id"`
		Stat         RouterStatistics      `json:"stat_total"`
		StatSpeed    RouterSpeedStatistics `json:"stat_in_second"`
		Addresses    []AddressInfo         `json:"addresses"`
	}

	c.mtx.Lock()
	var di DebugInfo
	di.AddressCount = len(c.addresses)
	di.NextMsgId = int(c.nextId)
	di.Stat = c.stat
	di.StatSpeed = c.statSpeed

	di.Addresses = make([]AddressInfo, 0, len(c.addresses))
	for address, a := range c.addresses {
		var ai AddressInfo
		ai.Address = address
		ai.MessageCount = a.MessagesCount()
		di.Addresses = append(di.Addresses, ai)
	}
	c.mtx.Unlock()

	sort.Slice(di.Addresses, func(i, j int) bool {
		return di.Addresses[i].Address < di.Addresses[j].Address
	})

	bsJson, _ := json.MarshalIndent(di, "", " ")
	c.mtx.Lock()
	c.lastDebugInfo = bsJson
	c.mtx.Unlock()

	return
}

const AddressBytesSize = 30
const AddressSize = int((AddressBytesSize * 8) / 5)

func CheckHash(hash []byte, complexity byte) bool {
	if len(hash) != 32 {
		return false
	}
	mask := make([]byte, (complexity/8)+1)
	for w := 0; w < len(mask); w++ {
		mask[w] = 0x00
	}

	for q := byte(0); q < complexity; q++ {
		byteIndex := int(q / 8)
		bitIndex := q % 8
		mask[byteIndex] = mask[byteIndex] | (0x80 >> bitIndex)
	}

	for k := 0; k < len(mask); k++ {
		if hash[k]&mask[k] != 0 {
			return false
		}
	}

	return true
}
