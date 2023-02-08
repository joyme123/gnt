package traceroute

import (
	"context"
	"net"
)

type TCPConn struct {
	IPv4 bool
	IPv6 bool
}

var _ Conn = &TCPConn{}

func NewTCPConn(ipv4, ipv6 bool) *TCPConn {
	u := &TCPConn{
		IPv4: ipv4,
		IPv6: ipv6,
	}
	return u
}

func (r *TCPConn) SendProbe(ctx context.Context, addr *net.IPAddr, srcPort, dstPort int, ttl uint8, data []byte) error {
	tcpConn, err := net.DialTCP(r.tcpNetwork(), &net.TCPAddr{
		Port: srcPort,
	}, &net.TCPAddr{
		IP:   addr.IP,
		Port: dstPort,
	})
	if err != nil {
		return err
	}

	defer tcpConn.Close()

	syscallConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}

	if err := SetTTL(syscallConn, ttl); err != nil {
		return err
	}

	_, err = tcpConn.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (r *TCPConn) tcpNetwork() string {
	network := "tcp"
	if r.IPv4 {
		network = "tcp4"
	} else if r.IPv6 {
		network = "tcp6"
	}

	return network
}
