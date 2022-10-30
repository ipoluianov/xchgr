package xchgr_server

import "net"

type Message struct {
	header MessageHeader
	data   []byte
}

type MessageHeader struct {
	id       int64
	sourceIP net.IP
	port     uint16
}

func NewMessage(id int64, sourceIP net.IP, port uint16, data []byte) *Message {
	var c Message
	c.header.id = id
	c.header.sourceIP = sourceIP
	c.header.port = port
	c.data = data
	return &c
}
