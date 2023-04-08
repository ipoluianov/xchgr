package name_client

import "errors"

type NameClient struct {
}

func NewNameClient() *NameClient {
	var c NameClient
	return &c
}

func (c *NameClient) GetAddressByName(name string) (string, error) {
	if name == "work.xchg" {
		return "#pem53ka2436w5bqgeaaqjud5uki4i7msbphqdezjehkz6ghp", nil
	}
	return "", errors.New("no address")
}
