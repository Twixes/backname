package server

import (
	"fmt"
	"net"
	"strconv"
)

func parseIPv4(parts []string) (net.IP, error) {
	if len(parts) != 4 {
		return nil, fmt.Errorf("IPv4 addresses must have 4 parts")
	}
	address := make(net.IP, 4)
	for i, part := range parts {
		octet, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return nil, err
		}
		address[i] = byte(octet)
	}
	return address, nil
}

func parseIPv6(parts []string) (net.IP, error) {
	if len(parts) != 8 {
		return nil, fmt.Errorf("IPv6 addresses must have 8 parts")
	}
	address := make(net.IP, 16)
	for i, part := range parts {
		octet, err := strconv.ParseUint(part, 16, 16)
		if err != nil {
			return nil, err
		}
		address[2*i] = byte(octet >> 8)
		address[2*i+1] = byte(octet)
	}
	return address, nil
}
