//go:build linux || darwin
// +build linux darwin

package traceroute

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func SetTTL(conn syscall.RawConn, ttl uint8) error {
	var err error
	if e := conn.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, int(ttl))
	}); e != nil {
		return e
	}

	return err
}
