package utils

import (
	"net"
	"strings"
)

func IsLocalHost(addr string) bool {

	if val, _, err := net.SplitHostPort(addr); err == nil {
		addr = val
	}

	return addr == "localhost" || net.ParseIP(addr).IsLoopback()
}

func IsLocalNetwork(addr string) bool {

	if val, _, err := net.SplitHostPort(addr); err == nil {
		addr = val
	}

	return strings.HasSuffix(addr, ".local") || net.ParseIP(addr).IsPrivate()
}
