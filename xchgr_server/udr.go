package xchgr_server

import (
	"encoding/json"
	"net"
	"sort"
	"sync"

	"github.com/ipoluianov/gomisc/logger"
)

type Udr struct {
	mtx      sync.Mutex
	db       map[string]string
	stopping bool
}

type UdrRecord struct {
	XchgAddress string
	IpPoint     string
}

type UdrState struct {
	Items []UdrRecord
}

func NewUdr() *Udr {
	var c Udr
	c.db = make(map[string]string)
	return &c
}

func (c *Udr) Start() {
	go c.th()
}

func (c *Udr) Stop() {
	c.stopping = true
}

func (c *Udr) State() string {
	addrs := make([]string, 0)
	c.mtx.Lock()
	defer c.mtx.Unlock()
	for key := range c.db {
		addrs = append(addrs, key)
	}
	sort.Slice(addrs, func(i, j int) bool {
		return addrs[i] < addrs[j]
	})
	var state UdrState
	for _, a := range addrs {
		var item UdrRecord
		item.XchgAddress = a
		item.IpPoint = c.db[a]
		state.Items = append(state.Items, item)
	}
	result, _ := json.MarshalIndent(state, "", " ")
	return string(result)
}

func (c *Udr) GetIPByXchgAddress(xchgAddress string) string {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	ip, ok := c.db[xchgAddress]
	if ok {
		return ip
	}
	ip, ok = c.db["#"+xchgAddress]
	if ok {
		return ip
	}
	return ""
}

func (c *Udr) th() {
	addr, _ := net.ResolveUDPAddr("udp", ":8585")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		logger.Println("ERROR:", err)
		return
	}

	for !c.stopping {
		buffer := make([]byte, 1024)
		bytesRead, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			panic(err)
		}
		incoming := string(buffer[0:bytesRead])
		c.mtx.Lock()
		c.db[incoming] = remoteAddr.String()
		c.mtx.Unlock()
	}

	if conn != nil {
		conn.Close()
	}

}
