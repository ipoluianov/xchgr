package xchgr_server

type Message struct {
	id   uint64
	data []byte
}

func NewMessage(id uint64, data []byte) *Message {
	var c Message
	c.id = id
	c.data = data
	return &c
}
