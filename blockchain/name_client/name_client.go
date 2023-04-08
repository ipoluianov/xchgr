package name_client

type NameClient struct {
}

func NewNameClient() *NameClient {
	var c NameClient
	return &c
}

func (c *NameClient) GetAddressByName(name string) (string, error) {
	return "-TODO-ADDRESS-", nil
}
