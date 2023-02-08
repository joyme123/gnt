//go:build windows || darwin
// +build windows darwin

package traceroute

import (
	"context"
	"fmt"
	"net"
)

// TCPHalfOpenConn send sync to remote host, if remote host response with ack, then send rst.
// so tcp connect can't be established. this ensures sending probe to remote host doesn't make
// side effect.
type TCPHalfOpenConn struct {
	IPv4 bool
	IPv6 bool
}

var _ Conn = &TCPHalfOpenConn{}

func NewTCPHalfOpenConn(ipv4, ipv6 bool) *TCPHalfOpenConn {
	u := &TCPHalfOpenConn{
		IPv4: ipv4,
		IPv6: ipv6,
	}
	return u
}

func (r *TCPHalfOpenConn) SendProbe(ctx context.Context, addr *net.IPAddr, srcPort, dstPort int, ttl uint8, _ []byte) error {
	fmt.Println("not implemented yet")
	return nil
}
