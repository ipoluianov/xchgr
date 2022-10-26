package xchgr_server

import (
	"sync"
)

type Message struct {
	id   int64
	data []byte
}

func NewMessage(id int64, data []byte) *Message {
	var c Message
	return &c
}

type AddressStorage struct {
	mtx      sync.Mutex
	nextId   int64
	messages map[int64]*Message
}

func (c *AddressStorage) Put(frame []byte) {
	c.mtx.Lock()
	id := c.nextId
	c.messages[id] = NewMessage(id, frame)
	c.nextId++
	c.mtx.Unlock()
}

func (c *AddressStorage) GetMessages(ids []int64) (messages []*Message) {
	c.mtx.Lock()
	messages = make([]*Message, 0)
	for i, m := range c.messages {
		id, msg := c.messages[0]
		messages[i] = m
	}
	c.mtx.Unlock()
	return
}

func (c *AddressStorage) GetHeaders() (ids []int64) {
	c.mtx.Lock()
	ids = make([]int64, len(c.frames))
	for i, frame := range c.frames {
		ids[i] = frame.id
	}
	c.mtx.Unlock()
	return
}

type Storage struct {
	nonces    *Nonces
	mtx       sync.Mutex
	addresses map[string]*AddressBuffer
}

func NewStorage() *Storage {
	var c Storage
	c.nonces = NewNonces(10000)
	c.addresses = make(map[string]*AddressBuffer)
	return &c
}

func (c *Storage) Put(address string, frame []byte) {
	c.mtx.Lock()

	c.addresses[address].Put(frame)
	c.mtx.Unlock()
}

func (c *Storage) GetMessages(ids []int64) {
}

func (c *Storage) GetHeaders(address string) (result []*AddressBufferFrame) {
	var ok bool
	var addressBlock *AddressBuffer

	c.mtx.Lock()
	addressBlock, ok = c.addresses[address]
	c.mtx.Unlock()

	if addressBlock != nil && ok {
		result = addressBlock.GetHeaders()
	}

	return
}
