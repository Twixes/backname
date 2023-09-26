package server

import (
	"net"
	"strings"
)

func parseIPv4Subdomain(subdomain string) net.IP {
	if strings.Contains(subdomain, "-") {
		subdomain = strings.ReplaceAll(subdomain, "-", ".")
	}
	address := net.ParseIP(subdomain)
	if address.To4() == nil { // Ensure not IPv6 address
		return nil
	}
	return address
}

func parseIPv6Subdomain(subdomain string) net.IP {
	if strings.Contains(subdomain, "-") {
		subdomain = strings.ReplaceAll(subdomain, "-", ":")
	} else {
		subdomain = strings.ReplaceAll(subdomain, ".", ":")
	}
	address := net.ParseIP(subdomain)
	return address
}
