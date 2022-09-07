package xchgr

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ipoluianov/gomisc/crypt_tools"
	"github.com/ipoluianov/xchg/xchg"
)

type Server struct {
	// Sync
	mtx sync.Mutex

	// State
	started  bool
	stopping bool

	// Data
	nonces *Nonces
	blocks map[string]*Block
}

type Block struct {
	data []byte
	dt   time.Time
}

const (
	UDP_PORT          = 8484
	NONCE_COUNT       = 1024 * 1024
	INPUT_BUFFER_SIZE = 1024 * 1024
	STORING_TIMEOUT   = 60 * time.Second
)

func NewServer() *Server {
	var c Server
	return &c
}

func (c *Server) Start() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// Checks
	if c.started {
		return errors.New("already started")
	}
	if c.stopping {
		return errors.New("it is stopping")
	}

	// Initialization
	c.blocks = make(map[string]*Block)
	c.nonces = NewNonces(NONCE_COUNT)

	// Start worker
	go c.thReceive()

	return nil
}

func (c *Server) Stop() error {
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
	c.blocks = nil
	c.nonces = nil
	c.mtx.Unlock()

	return nil
}

func (c *Server) thReceive() {
	var err error
	var conn net.PacketConn

	fmt.Println("started")

	c.started = true
	buffer := make([]byte, INPUT_BUFFER_SIZE)

	conn, err = net.ListenPacket("udp", ":"+fmt.Sprint(UDP_PORT))
	if err != nil {
		c.mtx.Lock()
		c.started = false
		c.stopping = false
		c.mtx.Unlock()
		fmt.Println("net.ListenPacket error:", err)
		return
	}

	var n int
	var addr net.Addr

	for {
		c.mtx.Lock()
		stopping := c.stopping
		c.mtx.Unlock()
		if stopping {
			break
		}

		err = conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
		if err != nil {
			c.mtx.Lock()
			c.started = false
			c.stopping = false
			c.mtx.Unlock()
			fmt.Println("conn.SetReadDeadline error:", err)
			return
		}

		n, addr, err = conn.ReadFrom(buffer)
		if errors.Is(err, os.ErrDeadlineExceeded) {
			c.backgroundOperations()
			continue
		}

		if err != nil {
			c.mtx.Lock()
			c.started = false
			c.stopping = false
			c.mtx.Unlock()
			fmt.Println("conn.ReadFrom error:", err)
			return
		}
		udpAddr, ok := addr.(*net.UDPAddr)
		if ok {
			frame := make([]byte, n)
			copy(frame, buffer[:n])
			go c.processFrame(conn, udpAddr, frame)
		} else {
			fmt.Println("unknown address type")
		}
	}

	fmt.Println("stoppped")
	c.mtx.Lock()
	c.started = false
	c.stopping = false
	c.mtx.Unlock()
}

func (c *Server) backgroundOperations() {
	c.mtx.Lock()
	now := time.Now()
	for key, value := range c.blocks {
		if value.dt.Sub(now) > STORING_TIMEOUT {
			delete(c.blocks, key)
		}
	}
	c.mtx.Unlock()
}

func (c *Server) processFrame(conn net.PacketConn, sourceAddress *net.UDPAddr, frame []byte) {
	fmt.Println("processFrame", sourceAddress, frame)
	if len(frame) < 8 {
		return
	}

	frameType := frame[0]
	switch frameType {
	case 0x00:
		c.processFrame00(conn, sourceAddress, frame)
	case 0x01:
		c.processFrame01(conn, sourceAddress, frame)
	case 0x02:
		c.processFrame02(conn, sourceAddress, frame)
	case 0x03:
		c.processFrame03(conn, sourceAddress, frame)
	}
}

func (c *Server) sendError(conn net.PacketConn, sourceAddress *net.UDPAddr, originalFrame []byte, errorCode byte) {
	var responseFrame [8]byte
	copy(responseFrame[:], originalFrame[:8])
	responseFrame[1] = errorCode
	_, _ = conn.WriteTo(responseFrame[:], sourceAddress)
}

func (c *Server) sendResponse(conn net.PacketConn, sourceAddress *net.UDPAddr, originalFrame []byte, responseFrame []byte) {
	copy(responseFrame, originalFrame[:8])
	responseFrame[1] = 0x00
	_, _ = conn.WriteTo(responseFrame, sourceAddress)
}

func (c *Server) processFrame00(conn net.PacketConn, sourceAddress *net.UDPAddr, frame []byte) {
	c.sendResponse(conn, sourceAddress, frame, make([]byte, 8))
}

func (c *Server) processFrame01(conn net.PacketConn, sourceAddress *net.UDPAddr, frame []byte) {
	response := make([]byte, 8+16)
	nonce := c.nonces.Next()
	copy(response[8:], nonce[:])
	c.sendResponse(conn, sourceAddress, frame, response)
}

func (c *Server) processFrame02(conn net.PacketConn, sourceAddress *net.UDPAddr, frame []byte) {
	/*
		8: nonce. 16 bytes
		24: salt. 8 bytes
		32: rsa-2048 signature. 256 bytes
		288: public key len. 4 bytes (PKL)
		292: public key
		292+PKL: data
	*/
	if len(frame) < 292 {
		c.sendError(conn, sourceAddress, frame, 0x01)
		return
	}
	nonce := frame[8:24]
	signature := frame[32:288]
	publicKeyLen := int(binary.LittleEndian.Uint32(frame[288:]))
	if len(frame) < 292+publicKeyLen {
		c.sendError(conn, sourceAddress, frame, 0x02)
		return
	}
	dataLen := len(frame) - 292 - publicKeyLen
	publicKeyBS := frame[292 : 292+publicKeyLen]
	data := frame[292+publicKeyLen:]

	// Check Nonce
	if !c.nonces.Check(nonce) {
		c.sendError(conn, sourceAddress, frame, 0x03)
		return
	}

	// Check hash
	hash := sha256.Sum256(frame[8:32]) // Nonce + Salt
	if !CheckHash(hash[:], frame[5]) {
		c.sendError(conn, sourceAddress, frame, 0x04)
		return
	}

	if dataLen > 256 {
		c.sendError(conn, sourceAddress, frame, 0x05)
		return
	}

	var publicKey *rsa.PublicKey
	var err error
	publicKey, err = crypt_tools.RSAPublicKeyFromDer(publicKeyBS)
	if err != nil {
		c.sendError(conn, sourceAddress, frame, 0x06)
		return
	}

	fmt.Println("Signature:", signature)

	hashForSign := sha256.Sum256(frame[8:32])

	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashForSign[:], signature)
	if err != nil {
		c.sendError(conn, sourceAddress, frame, 0x07)
		return
	}

	peerAddress := "#" + xchg.AddressForPublicKeyBS(publicKeyBS)
	c.mtx.Lock()
	existingBlock, existingBlockOk := c.blocks[peerAddress]
	if existingBlockOk {
		existingBlock.data = data
		existingBlock.dt = time.Now()
	} else {
		c.blocks[peerAddress] = &Block{data: data, dt: time.Now()}
	}
	c.mtx.Unlock()

	response := make([]byte, 8)
	c.sendResponse(conn, sourceAddress, frame, response)
}

func (c *Server) processFrame03(conn net.PacketConn, sourceAddress *net.UDPAddr, frame []byte) {
	addressBS := frame[8:]
	address := string(addressBS)
	nativeAddress, err := c.resolveAddress(address)
	if err != nil {
		fmt.Println(err)
		c.sendError(conn, sourceAddress, frame, 0x01)
		return
	}

	c.mtx.Lock()
	dataBlock, ok := c.blocks[nativeAddress]
	c.mtx.Unlock()

	if ok && dataBlock != nil {
		response := make([]byte, 8+len(addressBS)+1+len(dataBlock.data))
		copy(response[8:], addressBS)
		response[8+len(addressBS)] = '='
		copy(response[8+len(addressBS)+1:], dataBlock.data)
		c.sendResponse(conn, sourceAddress, frame, response)
	} else {
		c.sendError(conn, sourceAddress, frame, 0x02)
	}
}

func (c *Server) resolveAddress(address string) (nativeAddress string, err error) {
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
