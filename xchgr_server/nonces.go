package xchgr_server

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
)

//////////////////////////////////////////////////////
// Description:
// Size: 16 bytes
// [I][I][I][I] [C][R][R][R] [R][R][R][R] [R][R][R][R]
// I = Index in array for fast search
// C = complexity for PoW
// R = random
//////////////////////////////////////////////////////

const (
	NONCE_SIZE           = 16
	NONCE_COMPLEXITY_POS = 4
)

type Nonces struct {
	mtx          sync.Mutex
	nonces       [][NONCE_SIZE]byte
	currentIndex int
	complexity   byte
}

func NewNonces(size int) *Nonces {
	var c Nonces
	if size < 1 {
		size = 1 // Minimal size
	}
	c.complexity = 0
	c.nonces = make([][NONCE_SIZE]byte, size)
	for i := 0; i < size; i++ {
		c.fillNonce(i)
	}
	c.currentIndex = 0
	return &c
}

func (c *Nonces) fillNonce(index int) {
	if index >= 0 && index < len(c.nonces) {
		binary.LittleEndian.PutUint32(c.nonces[index][:], uint32(index)) // Index of nonce for search (4 bytes)
		c.nonces[index][NONCE_COMPLEXITY_POS] = c.complexity             // Current Complexity (1 byte)
		rand.Read(c.nonces[index][NONCE_COMPLEXITY_POS+1:])              // Random Nonce (11 bytes)
	}
}

func (c *Nonces) Next() [NONCE_SIZE]byte {
	var result [NONCE_SIZE]byte
	c.mtx.Lock()
	c.fillNonce(c.currentIndex)
	result = c.nonces[c.currentIndex]
	c.currentIndex++
	if c.currentIndex >= len(c.nonces) {
		c.currentIndex = 0
	}
	c.mtx.Unlock()
	return result
}

func (c *Nonces) Check(nonce []byte) bool {
	if len(nonce) < NONCE_SIZE {
		return false
	}
	result := true
	c.mtx.Lock()
	index := int(binary.LittleEndian.Uint32(nonce[:]))
	if index >= 0 && index < len(c.nonces) {
		for i := 0; i < NONCE_SIZE; i++ {
			if c.nonces[index][i] != nonce[i] {
				result = false
				break
			}
		}
	} else {
		result = false
	}
	if result {
		c.fillNonce(index)
	}
	c.mtx.Unlock()
	return result
}
