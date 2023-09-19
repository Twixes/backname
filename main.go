package main

import (
	"log"

	"github.com/Twixes/backname/internal/server"
	"github.com/miekg/dns"
)

func main() {
	handler := new(server.DNSHandler)
	server := &dns.Server{
		Addr:      ":53",
		Net:       "udp",
		Handler:   handler,
		UDPSize:   65535,
		ReusePort: true,
	}

	log.Printf("DNS server listening on %s\n", server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
