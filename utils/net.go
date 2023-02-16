package utils

import "net"

func IPAddrString(addr net.Addr) string {
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		return udpAddr.IP.String()
	} else if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		return tcpAddr.IP.String()
	} else if ipAddr, ok := addr.(*net.IPAddr); ok {
		return ipAddr.IP.String()
	}

	return ""
}
