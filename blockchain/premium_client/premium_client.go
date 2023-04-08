package premium_client

type PremiumClient struct {
}

func NewPremiumClient() *PremiumClient {
	var c PremiumClient
	return &c
}

func (c *PremiumClient) isPremium() bool {
	return true
}
