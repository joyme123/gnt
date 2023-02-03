//go:build linux
// +build linux

package ping

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (p *Pinger) parseMessage(proto int, buf []byte) (*icmp.Message, error) {
	return icmp.ParseMessage(proto, buf)
}

func (p *Pinger) matchID(id int, replyID int) bool {
	if p.Unprivileged {
		return true
	}

	// only matmatchIch id with privileged mode
	return p.id == replyID
}

func (p *Pinger) setTTL(c *icmp.PacketConn) error {
	if p.ipProtocolVersion == 4 {
		if err := c.IPv4PacketConn().SetTTL(p.TTL); err != nil {
			return err
		}
		if err := c.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true); err != nil {
			return err
		}
	} else {
		if err := c.IPv6PacketConn().SetHopLimit(p.TTL); err != nil {
			return err
		}

		if err := c.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			return err
		}
	}

	return nil
}
