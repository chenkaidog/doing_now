package ip

import (
	"encoding/binary"
	"encoding/hex"
	"net"
	"runtime"
)

func IPv4() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil {
				return ip.IP.String()
			}
		}
	}

	return ""
}

func IPv4Hex() string {
	if runtime.GOOS == "windows" {
		return "00000000"
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			if ipv4 := ip.IP.To4(); ipv4 != nil {
				return hex.EncodeToString(ipv4)
			}
		}
	}

	return ""
}

func IPv4Int() uint32 {
	if runtime.GOOS == "windows" {
		return 0
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return 0
	}

	for _, addr := range addrs {
		if ip, ok := addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil {
				return binary.BigEndian.Uint32(ip.IP.To4())
			}
		}
	}

	return 0
}
