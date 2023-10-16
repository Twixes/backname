package server

import (
	"net"
	"strings"
)

func parseIPv4Subdomain(subdomain string) net.IP {
	subdomainParts := strings.Split(subdomain, ".")
	var possibleIPv4 string
	if strings.Contains(subdomainParts[len(subdomainParts)-1], "-") {
		possibleIPv4 = strings.ReplaceAll(subdomainParts[len(subdomainParts)-1], "-", ".")
	} else {
		if len(subdomainParts) < 4 {
			return nil
		}
		possibleIPv4 = strings.Join(subdomainParts[len(subdomainParts)-4:], ".")
	}
	address := net.ParseIP(possibleIPv4)
	if address.To4() == nil { // Ensure not IPv6 address
		return nil
	}
	return address
}

func parseIPv6Subdomain(subdomain string) net.IP {
	subdomainParts := strings.Split(subdomain, ".")
	var possibleIPv6 string
	if strings.Contains(subdomainParts[len(subdomainParts)-1], "-") {
		possibleIPv6 = strings.ReplaceAll(subdomainParts[len(subdomainParts)-1], "-", ":")
	} else {
		if len(subdomainParts) < 8 {
			return nil
		}
		possibleIPv6 = strings.Join(subdomainParts[len(subdomainParts)-8:], ":")
	}
	address := net.ParseIP(possibleIPv6)
	return address
}
