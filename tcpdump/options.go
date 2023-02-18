package tcpdump

type Options struct {
	// Interface specifies the network interface to capture packets
	Interface string
	// Ethernet specifies whether to show ethernet info when dump packets
	Ethernet bool
}
