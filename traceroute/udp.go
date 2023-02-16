package traceroute

import (
	"context"
	"net"
)

type UDPConn struct {
	IPv4 bool
	IPv6 bool
}

var _ Conn = &UDPConn{}

func NewUDPConn(ipv4, ipv6 bool) *UDPConn {
	u := &UDPConn{
		IPv4: ipv4,
		IPv6: ipv6,
	}
	return u
}

func (r *UDPConn) SendProbe(ctx context.Context, addr *net.IPAddr, srcPort, dstPort int, ttl uint8, data []byte) error {
	udpConn, err := net.DialUDP(r.udpNetwork(), &net.UDPAddr{
		Port: srcPort,
	}, &net.UDPAddr{
		IP:   addr.IP,
		Port: dstPort,
	})
	if err != nil {
		return err
	}

	defer udpConn.Close()

	syscallConn, err := udpConn.SyscallConn()
	if err != nil {
		return err
	}

	if err := SetTTL(syscallConn, ttl); err != nil {
		return err
	}

	_, err = udpConn.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (r *UDPConn) udpNetwork() string {
	network := "udp"
	if r.IPv4 {
		network = "udp4"
	} else if r.IPv6 {
		network = "udp6"
	}

	return network
}
