package xchgr_server

import (
	"sync"
)

type AddressStorage struct {
	mtx         sync.Mutex
	maxMessages int
	messages    []*Message
}

func NewAddressStorage() *AddressStorage {
	var c AddressStorage
	c.maxMessages = 10000
	c.messages = make([]*Message, 0, c.maxMessages+1)
	return &c
}

func (c *AddressStorage) Put(id uint64, frame []byte) {
	c.mtx.Lock()
	msg := NewMessage(id, frame)
	c.messages = append(c.messages, msg)
	if len(c.messages) > c.maxMessages {
		c.messages = c.messages[1:]
		//fmt.Println("remove message-----------")
	}
	c.mtx.Unlock()
}

func (c *AddressStorage) GetMessage(afterId uint64, maxSize uint64) (data []byte, lastId uint64) {
	data = make([]byte, 0)
	lastId = afterId
	c.mtx.Lock()
	for _, m := range c.messages {
		if m.id > afterId {
			if len(data)+len(m.data) < int(maxSize) {
				data = append(data, m.data...)
				lastId = m.id
			}
		}
	}

	if len(data) == 0 && len(c.messages) > 0 {
		if afterId > c.messages[len(c.messages)-1].id {
			lastId = c.messages[0].id
		}
	}
	c.mtx.Unlock()
	return
}
