package client

type SouthboundCli struct {
}

func NewSouthboundCli() *SouthboundCli {
	return &SouthboundCli{}
}

func (south *SouthboundCli) Poll() error {
	return nil
}
