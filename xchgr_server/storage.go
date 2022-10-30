package xchgr_server

import (
	"errors"
	"net"
	"sync"
)

type Storage struct {
	nonces    *Nonces
	mtx       sync.Mutex
	addresses map[string]*AddressStorage
}

func NewStorage() *Storage {
	var c Storage
	c.nonces = NewNonces(10000)
	c.addresses = make(map[string]*AddressStorage)
	return &c
}

func (c *Storage) Put(addressDest string, sourceIP net.IP, port int, frame []byte) {
	var ok bool
	var addressStorage *AddressStorage

	c.mtx.Lock()
	addressStorage, ok = c.addresses[addressDest]
	if !ok || addressStorage == nil {
		addressStorage = NewAddressStorage()
		c.addresses[addressDest] = addressStorage
	}
	c.mtx.Unlock()

	addressStorage.Put(sourceIP, port, frame)
}

func (c *Storage) GetMessages(addressDest string, ids []int64) (result []*Message, err error) {
	result = make([]*Message, 0)
	var ok bool
	var addressStorage *AddressStorage

	c.mtx.Lock()
	addressStorage, ok = c.addresses[addressDest]
	c.mtx.Unlock()

	if !ok || addressStorage == nil {
		err = errors.New("no inbox found")
		return
	}

	result = addressStorage.GetMessages(ids)

	return
}

func (c *Storage) GetHeaders(addressDest string) (result []*MessageHeader, err error) {
	result = make([]*MessageHeader, 0)
	var ok bool
	var addressStorage *AddressStorage

	c.mtx.Lock()
	addressStorage, ok = c.addresses[addressDest]
	c.mtx.Unlock()

	if !ok || addressStorage == nil {
		err = errors.New("no inbox found")
		return
	}

	result = addressStorage.GetHeaders()

	return
}
