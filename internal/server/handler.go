package server

import (
	"log"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

var (
	zone                 = strings.ToLower(os.Getenv("ZONE"))
	websiteIPv4s         []net.IP
	websiteIPv6s         []net.IP
	nameserverPublicIPv4 net.IP
)

func init() {
	if zone == "" {
		log.Fatal("ZONE environment variable must be set")
	}
	if !strings.HasSuffix(zone, ".") {
		zone += "."
	}

	if websiteIPv4sRaw := os.Getenv("WEBSITE_A"); websiteIPv4sRaw != "" {
		for _, websiteIPv4Raw := range strings.Split(websiteIPv4sRaw, ",") {
			if websiteIPv4, err := parseIPv4(strings.Split(websiteIPv4Raw, ".")); err == nil {
				websiteIPv4s = append(websiteIPv4s, websiteIPv4)
			} else {
				log.Fatalf("WEBSITE_A environment variable is invalid: %s", err)
			}
		}
	}
	if websiteIPv6sRaw := os.Getenv("WEBSITE_AAAA"); websiteIPv6sRaw != "" {
		for _, websiteIPv6Raw := range strings.Split(websiteIPv6sRaw, ",") {
			if websiteIPv6, err := parseIPv6(strings.Split(websiteIPv6Raw, ".")); err == nil {
				websiteIPv6s = append(websiteIPv6s, websiteIPv6)
			} else {
				log.Fatalf("WEBSITE_AAAA environment variable is invalid: %s", err)
			}
		}
	}

	if nameserverPublicIPv4Raw := os.Getenv("NAMESERVER_PUBLIC_IPV4"); nameserverPublicIPv4Raw != "" {
		var err error
		nameserverPublicIPv4, err = parseIPv4(strings.Split(nameserverPublicIPv4Raw, "."))
		if err != nil {
			log.Fatal("NAMESERVER_PUBLIC_IPV4 environment variable is invalid")
		}
	} else {
		log.Fatal("NAMESERVER_PUBLIC_IPV4 environment variable must be set")
	}
}

type DNSHandler struct{}

// Resolve a question into an answer, an extra record and a response code
func resolve(question dns.Question) ([]dns.RR, int) {
	log.Printf("Resolving %s records for %s\n", dns.TypeToString[question.Qtype], question.Name)

	// Make sure that the name from the question lies within the zone
	if !strings.HasSuffix(strings.ToLower(question.Name), zone) {
		return nil, dns.RcodeNotZone
	}

	subdomainsString := strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(question.Name), zone), ".")
	var subdomains []string
	if subdomainsString != "" {
		subdomains = strings.Split(subdomainsString, ".")
	}

	// Verify domain existence and determine records
	var records []dns.RR
	if len(subdomains) == 0 { // <zone>
		switch question.Qtype {
		case dns.TypeA:
			for _, websiteIPv4 := range websiteIPv4s {
				records = append(records, &dns.A{
					A: websiteIPv4,
				})
			}
		case dns.TypeAAAA:
			for _, websiteIPv6 := range websiteIPv6s {
				records = append(records, &dns.AAAA{
					AAAA: websiteIPv6,
				})
			}
		}
	} else if len(subdomains) == 1 && subdomains[0] == "www" { // www.<zone>
		switch question.Qtype {
		case dns.TypeCNAME:
			records = append(records, &dns.CNAME{
				Target: zone,
			})
		}
	} else if len(subdomains) == 1 && subdomains[0] == "ns" { // ns.<zone>
		switch question.Qtype {
		case dns.TypeA:
			records = append(records, &dns.A{
				A: nameserverPublicIPv4,
			})
		}
	} else if subdomainIPv4, _ := parseIPv4(subdomains); subdomainIPv4 != nil { // <ipv4>.<zone>
		switch question.Qtype {
		case dns.TypeA:
			records = append(records, &dns.A{
				A: subdomainIPv4,
			})
		}
	} else if subdomainIPv6, _ := parseIPv6(subdomains); subdomainIPv6 != nil { // <ipv6>.<zone>
		switch question.Qtype {
		case dns.TypeAAAA:
			records = append(records, &dns.AAAA{
				AAAA: subdomainIPv6,
			})
		}
	} else {
		return nil, dns.RcodeNameError
	}

	if question.Qtype == dns.TypeNS { // Any domain in zone
		records = append(records, &dns.NS{
			Ns: "ns." + zone,
		})
	}

	return records, dns.RcodeSuccess
}

func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	// Refuse if there are multiple question resource records
	if len(r.Question) != 1 {
		msg.SetRcode(r, dns.RcodeRefused)
		w.WriteMsg(msg)
		return
	}

	question := r.Question[0]
	answers, rcode := resolve(question)
	for _, answer := range answers {
		header := answer.Header() // Fill in header boilerplate
		header.Class = dns.ClassINET
		header.Name = question.Name
		header.Ttl = 0 // TODO
		switch question.Qtype {
		case dns.TypeA:
			header.Rrtype = dns.TypeA
		case dns.TypeAAAA:
			header.Rrtype = dns.TypeAAAA
		case dns.TypeCNAME:
			header.Rrtype = dns.TypeCNAME
		case dns.TypeNS:
			header.Rrtype = dns.TypeNS
		}
		msg.Answer = append(msg.Answer, answer)
	}
	msg.SetRcode(r, rcode)

	w.WriteMsg(msg)
}
