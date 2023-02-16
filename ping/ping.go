package ping

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sync/errgroup"
)

// Pinger is an implement of ping command
type Pinger struct {
	// Count is times to send icmp/udp packets
	Count int
	// Interval is the interval to send packets
	Interval int
	// Interface is the network interface to send packets
	Interface string
	// Timestamp indicates whether to print timestamp before each line
	Timestamp bool
	// Quite output. Nothing is displayed except the summary lines at startup time and when finished.
	Quite bool
	// TTL set the IP time to live
	TTL int
	// Timeout is the total seconds to wait for sending packets
	Timeout int
	// Network options: ip(select automatically), ip4, ip6
	Network string

	Deadline int

	// TargetAddr is the target host address
	TargetAddr string

	// FIXME(jpf): can't receive ttl exceeded response on linux
	Unprivileged bool

	log         *log.Logger
	debugLogger *logr.Logger

	// resolvedTargetAddr ...
	resolvedTargetAddr *net.IPAddr

	// ipProtocolVersion: 4 or 6
	ipProtocolVersion int

	id       int
	sequence int

	sendPackets          int64
	receivePackets       int64
	minLatency           int
	maxLatency           int
	avgLatency           int
	sumOfSquareLatency   int64
	firstPacketTimestamp time.Time

	OnReceiveEchoReply              func(rm *icmp.Message, n int, ip net.Addr, ttl int)
	OnReceiveTTLExceeded            func(rm *icmp.Message, ip net.Addr)
	OnReceiveDestinationUnreachable func(rm *icmp.Message, ip net.Addr)
}

func (p *Pinger) Run(ctx context.Context) error {
	c, err := p.Listen(ctx)
	if err != nil {
		return err
	}

	if err := p.setTTL(c); err != nil {
		return err
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	var g errgroup.Group
	g.Go(func() error {
		defer cancel()
		return p.Receive(ctx, c)
	})
	g.Go(func() error {
		defer cancel()
		return p.Send(ctx, c)
	})
	err = g.Wait()
	p.printSummary()
	return err
}

func (p *Pinger) Listen(ctx context.Context) (*icmp.PacketConn, error) {
	if err := p.initDefaultOptions(); err != nil {
		return nil, err
	}

	network, err := p.network()
	if err != nil {
		return nil, err
	}

	listenAddr, err := p.getAddrByInterface()
	if err != nil {
		return nil, err
	}

	p.debugLogger.V(4).Info("listen", "nerwork", network, "addr", listenAddr)
	c, err := icmp.ListenPacket(network, listenAddr)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (p *Pinger) Send(ctx context.Context, c *icmp.PacketConn) error {
	interval := time.NewTicker(time.Duration(p.Interval) * time.Second)
	defer interval.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-interval.C:
			if !p.continueToPing() {
				return nil
			}

			if p.Deadline > 0 {
				if err := c.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(p.Deadline))); err != nil {
					return err
				}
			}

			icmpMessage := make([]byte, 0, 56)
			timeBytes := timeToBytes(time.Now())
			icmpMessage = append(icmpMessage, timeBytes...)
			for i := 0x08; i < 48+0x08; i++ {
				icmpMessage = append(icmpMessage, uint8(i))
			}

			wm := icmp.Message{
				Code: 0,
				Body: &icmp.Echo{
					ID:   p.id,
					Seq:  p.sequence,
					Data: icmpMessage,
				},
			}
			if p.ipProtocolVersion == 4 {
				wm.Type = ipv4.ICMPTypeEcho
			} else {
				wm.Type = ipv6.ICMPTypeEchoRequest
			}

			wb, err := wm.Marshal(nil)
			if err != nil {
				return err
			}

			var addr net.Addr
			addr = p.resolvedTargetAddr
			if p.Unprivileged {
				addr = &net.UDPAddr{
					IP: p.resolvedTargetAddr.IP,
				}
			}

			if _, err := c.WriteTo(wb, addr); err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					p.log.Printf("Request timeout for icmp_seq %d\n", p.sequence)
				} else {
					return err
				}
			}
			p.setSendMetrics()
		}
	}
}

func (p *Pinger) Receive(ctx context.Context, c *icmp.PacketConn) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			p.debugLogger.V(4).Info("start read packets from connection")
			if p.Deadline > 0 {
				if err := c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(p.Deadline))); err != nil {
					return err
				}
			}
			buf := make([]byte, 1500)
			var n int
			var ttl = -1
			var ip net.Addr
			var err error
			if p.ipProtocolVersion == 4 {
				var cm *ipv4.ControlMessage
				n, cm, ip, err = c.IPv4PacketConn().ReadFrom(buf)
				if cm != nil {
					ttl = cm.TTL
				}
			} else {
				var cm *ipv6.ControlMessage
				n, cm, ip, err = c.IPv6PacketConn().ReadFrom(buf)
				if cm != nil {
					ttl = cm.HopLimit
				}
			}
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					p.debugLogger.V(4).Info("read packets deadline exceeded", "msg", err.Error(), "n", n)
					time.Sleep(30 * time.Millisecond)
					continue
				} else {
					p.log.Printf("read failed: %v\n", err)
				}
				continue
			}

			p.debugLogger.V(4).Info("receive packet", "ip", ip, "bytes", n, "data", fmt.Sprintf("%x", buf))

			proto := 1 // icmp v4
			if p.ipProtocolVersion == 6 {
				proto = 58 // icmp v6
			}

			var rm *icmp.Message
			if rm, err = p.parseMessage(proto, buf); err != nil {
				return err
			}
			p.processICMPPacket(rm, n, ip, ttl)
		}
	}
}

func (p *Pinger) processICMPPacket(rm *icmp.Message, n int, ip net.Addr, ttl int) {
	// icmp: type(8), code(8), checksum(16), rest of header(32)
	p.debugLogger.V(4).Info("process icmp packet", "type", rm.Type)
	if p.ipProtocolVersion == 4 {
		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			p.OnReceiveEchoReply(rm, n, ip, ttl)
		case ipv4.ICMPTypeDestinationUnreachable:
			p.OnReceiveDestinationUnreachable(rm, ip)
		case ipv4.ICMPTypeTimeExceeded:
			p.OnReceiveTTLExceeded(rm, ip)
		default:
			p.debugLogger.V(4).Info("unknown packet type", "type", rm.Type, "message", rm)
		}
	} else {
		switch rm.Type {
		case ipv6.ICMPTypeEchoReply:
			p.OnReceiveEchoReply(rm, n, ip, ttl)
		case ipv6.ICMPTypeDestinationUnreachable:
			p.OnReceiveDestinationUnreachable(rm, ip)
		case ipv6.ICMPTypeTimeExceeded:
			p.OnReceiveTTLExceeded(rm, ip)
		default:
			p.debugLogger.V(4).Info("unknown packet type", "type", rm.Type, "message", rm)
		}

	}
}

func (p *Pinger) processEchoReply(rm *icmp.Message, n int, ip net.Addr, ttl int) {
	echo := rm.Body.(*icmp.Echo)
	p.debugLogger.V(4).Info("process echo reply", "ping id", p.id, "reply id", echo.ID)
	if !p.matchID(p.id, echo.ID) {
		return
	}

	sendTime := bytesToTime(echo.Data[0:8])
	timeMs := int(time.Now().Sub(sendTime).Milliseconds())
	if p.maxLatency == 0 || p.maxLatency < timeMs {
		p.maxLatency = timeMs
	}
	if p.minLatency == 0 || p.minLatency > timeMs {
		p.minLatency = timeMs
	}
	p.avgLatency = int(((p.receivePackets * int64(p.avgLatency)) + int64(timeMs)) / (p.receivePackets + 1))
	p.sumOfSquareLatency += int64(timeMs * timeMs)
	p.log.Printf("%d bytes from %s: icmp_seq=%d ttl=%d time=%d ms\n", n, ip, echo.Seq, ttl, timeMs)
	p.receivePackets++
}

func (p *Pinger) processDestinationUnreachable(rm *icmp.Message, ip net.Addr) {
	msg := rm.Body.(*icmp.DstUnreach)
	var icmpData []byte
	if p.ipProtocolVersion == 4 {
		icmpData = msg.Data[ipv4.HeaderLen:]
	} else {
		icmpData = msg.Data[ipv6.HeaderLen:]
	}

	p.debugLogger.V(4).Info("", "data", fmt.Sprintf("%x", icmpData), "id", fmt.Sprintf("%x", icmpData[4:6]), "seq", icmpData[6:8])

	id := binary.BigEndian.Uint16(icmpData[4:6])
	seq := binary.BigEndian.Uint16(icmpData[6:8])
	if !p.matchID(p.id, int(id)) {
		return
	}

	p.log.Printf("From %s icmp_seq=%d Destination unreachable", ip, seq)
}

func (p *Pinger) processTTLExceeded(rm *icmp.Message, ip net.Addr) {
	msg := rm.Body.(*icmp.TimeExceeded)
	var icmpData []byte
	if p.ipProtocolVersion == 4 {
		icmpData = msg.Data[ipv4.HeaderLen:]
	} else {
		icmpData = msg.Data[ipv6.HeaderLen:]
	}

	p.debugLogger.V(4).Info("", "data", fmt.Sprintf("%x", icmpData), "id", fmt.Sprintf("%x", icmpData[4:6]), "seq", icmpData[6:8])

	id := binary.BigEndian.Uint16(icmpData[4:6])
	seq := binary.BigEndian.Uint16(icmpData[6:8])
	if !p.matchID(p.id, int(id)) {
		return
	}

	p.log.Printf("From %s icmp_seq=%d Time to live exceeded", ip, seq)
}

func (p *Pinger) continueToPing() bool {
	if p.Timeout != 0 && !p.firstPacketTimestamp.IsZero() {
		if p.firstPacketTimestamp.Add(time.Second * time.Duration(p.Timeout)).Before(time.Now()) {
			return false
		}
	}

	if p.Count != 0 {
		if p.sendPackets >= int64(p.Count) {
			return false
		}
	}

	return true
}

func (p *Pinger) setSendMetrics() {
	if p.firstPacketTimestamp.IsZero() {
		p.firstPacketTimestamp = time.Now()
	}
	p.sendPackets++

	p.sequence++
	if float64(p.sequence) >= 65535 {
		p.sequence = 1
	}
}

func (p *Pinger) printSummary() {
	p.log.Printf("\n--- %s ping statistics ---\n", p.resolvedTargetAddr.IP.String())
	p.log.Printf("%d packets transmitted, %d packets received, %d%% packet loss\n",
		p.sendPackets, p.receivePackets, int(float64(p.sendPackets-p.receivePackets)/float64(p.sendPackets)*100))

	mdev := 0.0
	if p.receivePackets > 0 {
		mdev = math.Sqrt(float64((p.sumOfSquareLatency / p.receivePackets) - int64(p.avgLatency*p.avgLatency)))
	}
	p.log.Printf("round-trip min/avg/max/mdev = %d/%d/%d/%.3f ms\n", p.minLatency, p.avgLatency, p.maxLatency, mdev)
}

func (p *Pinger) initDefaultOptions() error {
	p.sequence = 1

	if err := p.resolveTargetAddr(); err != nil {
		return err
	}

	if p.resolvedTargetAddr.IP.To4() != nil {
		p.ipProtocolVersion = 4
	} else {
		p.ipProtocolVersion = 6
	}

	if p.Network == "" || p.Network == "ip" {
		if p.ipProtocolVersion == 4 {
			p.Network = "ip4"
		} else {
			p.Network = "ip6"
		}
	}

	p.id = os.Getpid() & 0xffff

	if p.log == nil {
		p.log = log.Default()
	}
	p.log.SetFlags(0)

	if p.OnReceiveEchoReply == nil {
		p.OnReceiveEchoReply = p.processEchoReply
	}
	if p.OnReceiveTTLExceeded == nil {
		p.OnReceiveTTLExceeded = p.processTTLExceeded
	}
	if p.OnReceiveDestinationUnreachable == nil {
		p.OnReceiveDestinationUnreachable = p.processDestinationUnreachable
	}

	return nil
}

func (p *Pinger) SetLogger(log *log.Logger) {
	p.log = log
}

func (p *Pinger) SetDebugLogger(log logr.Logger) {
	p.debugLogger = &log
}

func (p *Pinger) getAddrByInterface() (string, error) {
	if p.Interface == "" {
		if p.ipProtocolVersion == 4 {
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

	intf, err := net.InterfaceByName(p.Interface)
	if err != nil {
		return "", err
	}
	addrs, err := intf.Addrs()
	if err != nil {
		return "", err
	}

	if len(addrs) > 0 {
		ipnet := addrs[0].String()
		ip, _, err := net.ParseCIDR(ipnet)
		if err != nil {
			return "", err
		}

		if p.resolvedTargetAddr.IP.To4() != nil {
			// ipv4
			if ip.To4() != nil {
				return ip.To4().String(), nil
			}
		} else {
			if ip.To16() != nil {
				return ip.To16().String(), nil
			}
		}
	}

	return "", fmt.Errorf("interface %s doesn't have any ip address", p.Interface)
}

func (p *Pinger) resolveTargetAddr() error {
	if p.TargetAddr == "" {
		return fmt.Errorf("target address must be specified")
	}

	ip, err := net.ResolveIPAddr(p.Network, p.TargetAddr)
	if err != nil {
		return err
	}

	p.resolvedTargetAddr = ip

	return nil
}

func (p *Pinger) network() (string, error) {
	if p.Network != "" && p.Network != "ip4" && p.Network != "ip6" {
		return "", fmt.Errorf("network %s is not allowed, valid network: ip, ip4, ip6", p.Network)
	}

	network := ""

	if p.Unprivileged {
		if p.Network == "ip4" {
			network = "udp4"
		} else {
			network = "udp6"
		}
	} else {
		if p.Network == "ip4" {
			network = "ip4:icmp"
		} else {
			network = "ip6:ipv6-icmp"
		}
	}

	return network, nil
}

func timeToBytes(t time.Time) []byte {
	usec := int32(t.UnixMicro() % (1000 * 1000))
	sec := int32(t.UnixMicro() / (1000 * 1000))

	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, uint32(sec))
	binary.BigEndian.PutUint32(b[4:], uint32(usec))
	return b
}

func bytesToTime(b []byte) time.Time {
	sec := binary.BigEndian.Uint32(b[0:4])
	usec := binary.BigEndian.Uint32(b[4:])

	return time.Unix(int64(sec), int64(usec)*1000)
}
