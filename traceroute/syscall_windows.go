//go:build windows
// +build windows

package traceroute

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func SetTTL(conn syscall.RawConn, ttl int) error {
	var err error
	if e := conn.Control(func(fd uintptr) {
		err = windows.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl)
	}); e != nil {
		return e
	}

	return err
}
