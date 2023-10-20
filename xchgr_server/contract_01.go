package xchgr_server

import (
	"os"
	"time"

	"github.com/ipoluianov/gazer-billing-contract-eth/api"
	"github.com/ipoluianov/gomisc/logger"
	"github.com/kardianos/osext"
)

type Contract01 struct {
	shop     *api.Shop
	started  bool
	stopping bool

	counterSuccess int
	counterError   int
}

func NewContract01() *Contract01 {
	var c Contract01
	return &c
}

func (c *Contract01) Start() error {
	go c.tick()
	return nil
}

func (c *Contract01) tick() {
	c.started = true
	exePath, _ := osext.ExecutableFolder()
	err := os.MkdirAll(exePath+"/data/contract01", 0777)
	if err != nil {
		logger.Println("make data dir error:", err)
		c.started = false
		return
	}

	bsUrl, err := os.ReadFile(exePath + "/data/contract01/url.txt")
	if err != nil {
		logger.Println("read url.txt error:", err)
		c.started = false
		return
	}
	bsContractAddress, err := os.ReadFile(exePath + "/data/contract01/address.txt")
	if err != nil {
		logger.Println("read address.txt error:", err)
		c.started = false
		return
	}
	c.shop = api.NewShop(exePath+"/data/contract01/", string(bsUrl), string(bsContractAddress))
	c.shop.Load()
	err = c.shop.Update()
	if err != nil {
		logger.Println("Update contract01 error:", err)
		c.counterError++
	} else {
		c.counterSuccess++
	}

	dtOperationTime := time.Now().UTC()

	periodMs := 5000

	for !c.stopping {
		for {
			if c.stopping || time.Since(dtOperationTime) > time.Duration(periodMs)*time.Millisecond {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if c.stopping {
			break
		}
		dtOperationTime = time.Now().UTC()
		err = c.shop.Update()
		if err != nil {
			logger.Println("Shop Update ERROR:", err)
		}
	}
}

func (c *Contract01) Stop() {
	c.stopping = true
	time.Sleep(200 * time.Millisecond)
}

func (c *Contract01) IsPremium(xchgAddress string) bool {
	return c.shop.IsPremium(xchgAddress)
}

func (c *Contract01) CounterSuccess() int {
	return c.counterSuccess
}

func (c *Contract01) CounterError() int {
	return c.counterError
}

func (c *Contract01) RecordsCount() int {
	return c.shop.RecordsCount()
}

func (c *Contract01) Records() []api.ShopRecord {
	return c.shop.Records()
}
