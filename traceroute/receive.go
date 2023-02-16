package traceroute

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/joyme123/gnt/ping"
	"github.com/joyme123/gnt/utils"
)

func (r *TraceRouter) Receive(ctx context.Context, ch chan<- struct{}) error {
	network := "ip"
	if r.IPv4 {
		network = "ip4"
	} else if r.IPv6 {
		network = "ip6"
	}

	pinger := ping.Pinger{
		Network:                         network,
		Deadline:                        1,
		TargetAddr:                      r.DstAddr,
		Unprivileged:                    r.Unprivileged,
		OnReceiveEchoReply:              r.onReceiveEchoReply,
		OnReceiveTTLExceeded:            r.onReceiveTTLExceeded,
		OnReceiveDestinationUnreachable: r.onReceiveDestinationUnreachable,
	}
	pinger.SetDebugLogger(r.debugLogger)
	c, err := pinger.Listen(ctx)
	if err != nil {
		return err
	}

	ch <- struct{}{}
	r.debugLogger.V(4).Info("start receive packets")
	return pinger.Receive(ctx, c)
}

func (r *TraceRouter) onReceiveEchoReply(rm *icmp.Message, n int, ip net.Addr, ttl int) {
	msg, err := rm.Body.(*icmp.Echo).Marshal(0)
	if err != nil {
		return
	}
	r.processReceivePacket(msg, ip)
}

func (r *TraceRouter) onReceiveTTLExceeded(rm *icmp.Message, ip net.Addr) {
	msg := rm.Body.(*icmp.TimeExceeded)
	r.processReceivePacket(msg.Data, ip)
}

func (r *TraceRouter) onReceiveDestinationUnreachable(rm *icmp.Message, ip net.Addr) {
	msg := rm.Body.(*icmp.DstUnreach)
	r.processReceivePacket(msg.Data, ip)
}

func (r *TraceRouter) processReceivePacket(data []byte, ip net.Addr) {
	// https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xml
	var receivedDstIP net.IP
	// udp: src port, tcp: src port, icmp: id
	var receivedSrcIdentity int
	// udp/tcp: dst_port-start_dst_port, icmp: seq
	var receivedDstIdentity int
	var layer4Data []byte
	var receivedProtocol int

	if r.IPv4 {
		hdr, err := ipv4.ParseHeader(data[0:ipv4.HeaderLen])
		if err != nil {
			return
		}
		receivedDstIP = hdr.Dst
		receivedProtocol = hdr.Protocol
		layer4Data = data[ipv4.HeaderLen:]
	} else {
		hdr, err := ipv6.ParseHeader(data[0:ipv6.HeaderLen])
		if err != nil {
			return
		}
		receivedDstIP = hdr.Dst
		layer4Data = data[ipv6.HeaderLen:]
		receivedProtocol = hdr.NextHeader
	}

	if receivedProtocol == 17 && (r.method == "udp" || r.method == "default") {
		receivedSrcIdentity = int(binary.BigEndian.Uint16(layer4Data[0:2]))
		receivedDstIdentity = int(binary.BigEndian.Uint16(layer4Data[2:4])) - r.startPort
	} else if receivedProtocol == 6 && r.method == "tcp" {
		receivedSrcIdentity = int(binary.BigEndian.Uint16(layer4Data[0:2]))
		receivedDstIdentity = int(binary.BigEndian.Uint16(layer4Data[2:4])) - r.startPort
	} else if receivedProtocol == 1 && r.method == "icmp" {
		receivedSrcIdentity = int(binary.BigEndian.Uint16(layer4Data[4:6]))
		receivedDstIdentity = int(binary.BigEndian.Uint16(layer4Data[6:8]))
	}

	if receivedSrcIdentity != r.id() || receivedDstIP.String() != r.DstAddr {
		return
	}

	r.updateStatistic((uint8(receivedDstIdentity))/3+1, receivedDstIdentity%3, utils.IPAddrString(ip))
}

func (r *TraceRouter) updateStatistic(ttl uint8, index int, ip string) {

	// check timeout probe packets
	if ip == "" {
		for t, timestamps := range r.sendPacketsTimestamps {
			addrs, ok := r.statistics[t]
			if !ok || len(addrs) < 3 {
				for i, timestamp := range timestamps {
					if timestamp.Add(5 * time.Second).Before(time.Now()) {
						// timeout
						if r.statistics[t] == nil {
							r.statistics[t] = make(map[int]*PacketInfo)
						}
						if r.statistics[t][i] == nil {
							r.statistics[t][i] = &PacketInfo{
								IP:  "*",
								RTT: 0,
							}
						}
					}
				}
			}
		}
	} else {
		if r.statistics[ttl] == nil {
			r.statistics[ttl] = make(map[int]*PacketInfo)
		}
		r.statistics[ttl][index] = &PacketInfo{
			IP:  ip,
			RTT: int(time.Now().Sub(r.sendPacketsTimestamps[ttl][index]).Milliseconds()),
		}
	}
	r.printStatistic()
}

func (r *TraceRouter) printStatistic() {
	addrs, ok := r.statistics[r.printTTL]
	if !ok {
		return
	}

	if addrs[r.printAddr] != nil {
		if r.printAddr == 0 {
			fmt.Printf("%d  ", r.printTTL)
		}
		for i := r.printAddr; addrs[i] != nil; i++ {
			if addrs[i].IP == "*" {
				fmt.Printf("%s  ", addrs[i].IP)
			} else {
				fmt.Printf("%s %dms ", addrs[i].IP, addrs[i].RTT)
			}
			r.printAddr++
		}
		if len(addrs) == 3 {
			r.printTTL++
			r.printAddr = 0
			fmt.Println()
		} else {
			return
		}
	}
}
