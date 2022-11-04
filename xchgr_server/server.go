package xchgr_server

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base32"
	"encoding/binary"
	"errors"
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

func (c *Router) backgroundOperations() {
	c.mtx.Lock()
	c.mtx.Unlock()
}

func (c *Router) Put(frame []byte) {
	var ok bool
	var addressStorage *AddressStorage

	addressDestBS := frame[70:100]
	addressDest := "#" + base32.StdEncoding.EncodeToString(addressDestBS)
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
}

func (c *Router) processFrames(frames []byte) (response []byte, err error) {
	offset := 0

	for offset < len(frames) {
		if offset+128 <= len(frames) {
			frameLen := int(binary.LittleEndian.Uint32(frames[offset:]))
			if offset+frameLen <= len(frames) {
				response, err = c.processFrame(frames[offset : offset+frameLen])
				if err != nil {
					return
				}
			} else {
				break
			}
			offset += frameLen
		} else {
			break
		}
	}
	return
}

func (c *Router) processFrame(frame []byte) (response []byte, err error) {
	if len(frame) < 128 {
		return
	}
	//fmt.Println("processFrame", frame[8])

	frameType := frame[8]

	if frameType < 0x10 {
		switch frameType {
		case 0x00:
			response, err = c.processFrame00(frame)
		case 0x01:
			response, err = c.processFrame01(frame)
		case 0x02:
			response, err = c.processFrame02(frame)
		case 0x03:
			response, err = c.processFrame03(frame)
		case 0x04:
			response, err = c.processFrame04(frame)
		case 0x05:
			response, err = c.processFrame05(frame)
		case 0x06:
			response, err = c.processFrame06(frame)
		case 0x07:
			response, err = c.processFrame07(frame)
		case 0x08:
			response, err = c.processFrame08(frame)
		case 0x09:
			response, err = c.processFrame09(frame)
		}
	} else {

		c.Put(frame)
	}

	return
}

// Ping request
func (c *Router) processFrame00(frame []byte) (response []byte, err error) {
	response = make([]byte, len(frame))
	copy(response, frame)
	response[8] = 0x01
	return
}

// Ping response
func (c *Router) processFrame01(frame []byte) (response []byte, err error) {
	return
}

// GetNonce request
func (c *Router) processFrame02(frame []byte) (response []byte, err error) {
	response = make([]byte, 128+16)
	nonce := c.nonces.Next()
	copy(response[128:], nonce[:])
	return
}

// GetNonce response
func (c *Router) processFrame03(frame []byte) (response []byte, err error) {
	return
}

// Get messages headers request
func (c *Router) processFrame04(frame []byte) (response []byte, err error) {
	return
}

// Get messages headers response
func (c *Router) processFrame05(frame []byte) (response []byte, err error) {
	return
}

// Get message request
func (c *Router) processFrame06(frame []byte) (response []byte, err error) {
	var ok bool
	var addressStorage *AddressStorage

	if len(frame) < 128+8 {
		err = errors.New("wrong frame size")
		return
	}

	afterId := binary.LittleEndian.Uint64(frame[128+0:])
	maxSize := binary.LittleEndian.Uint64(frame[128+8:])

	addressSrcBS := frame[40:70]
	addressSrc := "#" + base32.StdEncoding.EncodeToString(addressSrcBS)

	c.mtx.Lock()
	addressStorage, ok = c.addresses[addressSrc]
	c.mtx.Unlock()

	if !ok || addressStorage == nil {
		return
	}

	msgData, lastId := addressStorage.GetMessage(afterId, maxSize)
	response = make([]byte, 8+len(msgData))
	binary.LittleEndian.PutUint64(response[0:], lastId)
	if msgData != nil {
		copy(response[8:], msgData)
	}

	return
}

func RSAPublicKeyFromDer(publicKeyDer []byte) (publicKey *rsa.PublicKey, err error) {
	publicKey, err = x509.ParsePKCS1PublicKey(publicKeyDer)
	return
}

// Get message response
func (c *Router) processFrame07(frame []byte) (response []byte, err error) {
	return
}

// Resolve name request
func (c *Router) processFrame08(frame []byte) (response []byte, err error) {
	return
}

// Resolve name response
func (c *Router) processFrame09(frame []byte) (response []byte, err error) {
	return
}

// Put call
func (c *Router) processFrame10(frame []byte) (response []byte, err error) {
	return
}

// Put answer
func (c *Router) processFrame11(frame []byte) (response []byte, err error) {
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
