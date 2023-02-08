package traceroute

import (
	"context"
	"net"
	"syscall"
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
	localAddr, err := r.getLocalAddr(addr)
	if err != nil {
		return err
	}
	dialer := net.Dialer{
		LocalAddr: &net.TCPAddr{
			IP:   net.ParseIP(localAddr),
			Port: srcPort,
		},
	}
	dialer.Control = func(network, address string, c syscall.RawConn) error {
		if err := SetTTL(c, ttl); err != nil {
			return err
		}
		return nil
	}
	conn, err := dialer.DialContext(ctx, r.tcpNetwork(), (&net.TCPAddr{
		IP:   addr.IP,
		Port: dstPort,
	}).String())
	if err != nil {
		return nil
	}

	defer conn.Close()

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

func (r *TCPConn) getLocalAddr(targetAddr *net.IPAddr) (string, error) {
	if targetAddr.IP.To4() != nil {
		// ipv4
		conn, err := net.Dial("udp", "1.1.1.1:80")
		if err != nil {
			return "", err
		}
		defer conn.Close()
		return conn.LocalAddr().(*net.UDPAddr).IP.To4().String(), nil
	}
	// ipv6
	conn, err := net.Dial("udp", "[2606:4700:4700::1111]:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.To16().String(), nil
}
