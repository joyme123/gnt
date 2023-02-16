package traceroute

import (
	"context"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

type Conn interface {
	SendProbe(ctx context.Context, addr *net.IPAddr, srcPort, dstPort int, ttl uint8, data []byte) error
}

type PacketInfo struct {
	IP  string
	RTT int
}

type TraceRouter struct {
	// if user use default tracing, port is default(33434)
	// if user specifies udp, port is 53
	// otherwise it is user specified port
	Port int
	IPv4 bool
	IPv6 bool
	// FirstTTL specifies with what TTL to start. Defaults to 1.
	FirstTTL uint8
	// MaxTTL specifies the maximum number of hops (max time-to-live value)
	// traceroute will probe. The default is 30.
	MaxTTL uint8
	// Squeries specifies the number of probe packets sent out simultaneously.
	// Sending several probes concurrently can speed up traceroute considerably.
	// The default value is 16.
	Squeries int
	// Sets the number of probe packets per hop. The default is 3.
	Nqueries int
	// Set the time (in seconds) to wait for a response to a probe (default 5.0 sec).
	WaitTime int
	// Minimal time interval between probes (default 0).
	// If the value is more than 10, then it specifies a number in milliseconds,
	// else it is a number of seconds (float point values allowed too).
	// Useful when some routers use rate-limit for icmp messages.
	SendWait int

	DstAddr string

	Unprivileged bool

	ttl       uint8
	conn      Conn
	method    string
	startPort int

	sendPacketsTimestamps map[uint8][]time.Time
	statistics            map[uint8]map[int]*PacketInfo

	printTTL  uint8
	printAddr int

	debugLogger logr.Logger
}

func NewTraceRouter(opt Options, dst string, debugLogger logr.Logger) *TraceRouter {
	r := &TraceRouter{
		DstAddr:               dst,
		sendPacketsTimestamps: make(map[uint8][]time.Time),
		statistics:            make(map[uint8]map[int]*PacketInfo),
		debugLogger:           debugLogger,
	}
	r.initDefaultOpts(opt)
	return r
}

func (r *TraceRouter) initDefaultOpts(opt Options) {
	if opt.Port > 0 {
		r.Port = opt.Port
	} else if opt.UDP {
		r.Port = 53
	} else {
		r.Port = 33434
	}
	r.startPort = r.Port

	r.IPv4 = opt.IPv4
	r.IPv6 = opt.IPv6
	r.FirstTTL = opt.FirstTTL
	r.MaxTTL = opt.MaxTTL
	r.Squeries = opt.Squeries
	r.Nqueries = opt.Nqueries

	r.WaitTime = 5
	if opt.WaitTime > 0 {
		r.WaitTime = opt.WaitTime
	}
	r.SendWait = opt.SendWait

	r.Unprivileged = opt.Unprivileged

	if opt.ICMP {
		r.method = "icmp"
	} else if opt.UDP {
		r.conn = NewUDPConn(r.IPv4, r.IPv6)
		r.method = "udp"
	} else if opt.TCP {
		r.method = "tcp"
		if runtime.GOOS == "linux" {
			r.debugLogger.V(4).Info("use tcp half open connection")
			r.conn = NewTCPHalfOpenConn(r.IPv4, r.IPv6)
		} else {
			r.conn = NewTCPConn(r.IPv4, r.IPv6)
		}
	} else {
		r.conn = NewUDPConn(r.IPv4, r.IPv6)
		r.method = "default"
	}

	r.ttl = r.FirstTTL
	r.printTTL = r.FirstTTL
	r.printAddr = 0
}

func (r *TraceRouter) Run(ctx context.Context) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	var g errgroup.Group

	c := make(chan struct{})

	g.Go(func() error {
		defer cancel()
		return r.Receive(ctx, c)
	})

	g.Go(func() error {
		defer cancel()
		<-c
		return r.Send(ctx)
	})

	err := g.Wait()

	r.printStatistic()
	return err
}

func (r *TraceRouter) Send(ctx context.Context) error {
	addr, err := r.resolvAddr()
	if err != nil {
		return err
	}

	if ok, err := r.sendProbe(ctx, addr); err != nil {
		return err
	} else if !ok {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if r.complete() {
				return nil
			}

			if !r.continueToRun() {
				r.updateStatistic(r.ttl, 0, "")
				time.Sleep(50 * time.Millisecond)
				continue
			}
			if ok, err := r.sendProbe(ctx, addr); err != nil {
				return err
			} else if !ok {
				return nil
			}
		}
	}
}

func (r *TraceRouter) continueToRun() bool {
	return len(r.statistics) == int(r.ttl-1) && len(r.statistics[r.ttl-1]) >= 3
}

func (r *TraceRouter) complete() bool {
	if len(r.statistics) == int(r.ttl-1) {
		for i := range r.statistics[r.ttl-1] {
			if r.statistics[r.ttl-1][i].IP == r.DstAddr {
				return true
			}
		}
	}
	return false
}

func (r *TraceRouter) sendProbe(ctx context.Context, addr *net.IPAddr) (bool, error) {
	for i := 0; i < 3; i++ {
		r.sendPacketsTimestamps[r.ttl] = append(r.sendPacketsTimestamps[r.ttl], time.Now())
		if err := r.conn.SendProbe(ctx, addr, r.id(), r.Port, r.ttl, []byte{0x00, uint8(i)}); err != nil {
			return false, err
		}
		r.Port++
		// send too fast will cause icmp drop
		time.Sleep(100 * time.Millisecond)
	}

	r.updateStatistic(r.ttl, 0, "")
	r.ttl++
	if r.ttl > r.MaxTTL {
		return false, nil
	}
	return true, nil
}

func (r *TraceRouter) resolvAddr() (*net.IPAddr, error) {
	network := "ip"
	if r.IPv4 {
		network = "ip4"
	} else if r.IPv6 {
		network = "ip6"
	}

	ipaddr, err := net.ResolveIPAddr(network, r.DstAddr)
	if err != nil {
		return nil, err
	}

	r.IPv4 = true
	r.IPv6 = false
	if ipaddr.IP.To4() == nil {
		r.IPv6 = true
		r.IPv4 = false
	}

	return ipaddr, nil
}

func (r *TraceRouter) id() int {
	return (os.Getpid() & 0xffff) | 0x8000
}
