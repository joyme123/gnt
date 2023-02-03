//go:build windows
// +build windows

package ping

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (p *Pinger) parseMessage(proto int, buf []byte) (*icmp.Message, error) {
	if p.Unprivileged {
		if p.ipProtocolVersion == 4 {
			buf = buf[ipv4.HeaderLen:]
		} else {
			buf = buf[ipv6.HeaderLen:]
		}
	}

	return icmp.ParseMessage(proto, buf)
}

func (p *Pinger) matchID(id int, replyID int) bool {
	return p.id == replyID
}

func (p *Pinger) setTTL(c *icmp.PacketConn) error {
	if p.ipProtocolVersion == 4 {
		if err := c.IPv4PacketConn().SetTTL(p.TTL); err != nil {
			return err
		}
	} else {
		if err := c.IPv6PacketConn().SetHopLimit(p.TTL); err != nil {
			return err
		}
	}

	return nil
}
