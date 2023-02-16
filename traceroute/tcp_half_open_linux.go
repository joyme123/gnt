//go:build linux
// +build linux

package traceroute

import (
	"context"
	"net"
	"time"

	"golang.org/x/sys/unix"
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
	pollerFd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return err
	}
	defer unix.Close(pollerFd)

	parsedAddr, family, err := r.parseSocket((&net.TCPAddr{
		IP:   addr.IP,
		Port: dstPort,
	}).String())
	if err != nil {
		return err
	}
	fd, err := unix.Socket(family, unix.SOCK_STREAM, 0)
	unix.CloseOnExec(fd)
	if err := unix.SetNonblock(fd, true); err != nil {
		return err
	}
	unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_TTL, int(ttl))
	unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_QUICKACK, 0)
	unix.SetsockoptLinger(fd, unix.SOL_SOCKET, unix.SO_LINGER, &unix.Linger{Onoff: 1, Linger: 0})
	defer unix.Close(fd)

	if family == unix.AF_INET {
		if err := unix.Bind(fd, &unix.SockaddrInet4{
			Port: srcPort,
		}); err != nil {
			return err
		}
	} else {
		if err := unix.Bind(fd, &unix.SockaddrInet6{
			Port: srcPort,
		}); err != nil {
			return err
		}
	}

	switch serr := unix.Connect(fd, parsedAddr); serr {
	case unix.EALREADY, unix.EINPROGRESS, unix.EINTR:
		break
	case unix.EISCONN, nil:
		return nil
	case unix.EINVAL:
		return serr
	default:
		return serr
	}

	// register events to epoll
	var event unix.EpollEvent
	event.Events = unix.EPOLLOUT | unix.EPOLLIN | unix.EPOLLET
	event.Fd = int32(fd)
	if err := unix.EpollCtl(pollerFd, unix.EPOLL_CTL_ADD, fd, &event); err != nil {
		return err
	}

	// poll events
	var epollEvents [32]unix.EpollEvent
	_, err = unix.EpollWait(pollerFd, epollEvents[:], int(time.Second.Milliseconds()))
	if err != nil {
		if err == unix.EINTR {
			return nil
		}
		return err
	}
	return nil
}

func (r *TCPHalfOpenConn) parseSocket(addr string) (sAddr unix.Sockaddr, family int, err error) {
	tAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}

	if ip := tAddr.IP.To4(); ip != nil {
		var addr4 [net.IPv4len]byte
		copy(addr4[:], ip)
		sAddr = &unix.SockaddrInet4{Port: tAddr.Port, Addr: addr4}
		family = unix.AF_INET
		return
	}

	if ip := tAddr.IP.To16(); ip != nil {
		var addr16 [net.IPv6len]byte
		copy(addr16[:], ip)
		sAddr = &unix.SockaddrInet6{Port: tAddr.Port, Addr: addr16}
		family = unix.AF_INET6
		return
	}

	err = &net.AddrError{
		Err:  "unsupported address family",
		Addr: tAddr.IP.String(),
	}
	return
}

func (r *TCPHalfOpenConn) tcpNetwork() string {
	network := "tcp"
	if r.IPv4 {
		network = "tcp4"
	} else if r.IPv6 {
		network = "tcp6"
	}

	return network
}
