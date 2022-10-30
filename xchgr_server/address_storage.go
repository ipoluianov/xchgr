package xchgr_server

import (
	"net"
	"sync"
)

type AddressStorage struct {
	mtx      sync.Mutex
	nextId   int64
	messages map[int64]*Message
}

func NewAddressStorage() *AddressStorage {
	var c AddressStorage
	return &c
}

func (c *AddressStorage) Put(sourceIP net.IP, port int, frame []byte) {
	c.mtx.Lock()
	id := c.nextId
	c.messages[id] = NewMessage(id, sourceIP, port, frame)
	c.nextId++
	c.mtx.Unlock()
}

func (c *AddressStorage) GetMessages(ids []int64) (messages []*Message) {
	c.mtx.Lock()
	messages = make([]*Message, 0)
	for i, m := range c.messages {
		messages[i] = m
	}
	c.mtx.Unlock()
	return
}

func (c *AddressStorage) GetHeaders() (result []*MessageHeader) {
	c.mtx.Lock()
	result = make([]*MessageHeader, len(c.messages))
	for i, m := range c.messages {
		result[i] = &m.header
	}
	c.mtx.Unlock()
	return
}
