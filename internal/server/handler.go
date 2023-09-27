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
	websiteA             []net.IP
	websiteAAAA          []net.IP
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
			if websiteIPv4 := net.ParseIP(websiteIPv4Raw); websiteIPv4 != nil {
				websiteA = append(websiteA, websiteIPv4)
			} else {
				log.Fatalf("WEBSITE_A environment variable is invalid: %s", websiteIPv4Raw)
			}
		}
	}
	if websiteIPv6sRaw := os.Getenv("WEBSITE_AAAA"); websiteIPv6sRaw != "" {
		for _, websiteIPv6Raw := range strings.Split(websiteIPv6sRaw, ",") {
			if websiteIPv6 := net.ParseIP(websiteIPv6Raw); websiteIPv6 != nil {
				websiteAAAA = append(websiteAAAA, websiteIPv6)
			} else {
				log.Fatalf("WEBSITE_AAAA environment variable is invalid: %s", websiteAAAA)
			}
		}
	}

	if nameserverPublicIPv4Raw := os.Getenv("NAMESERVER_PUBLIC_IPV4"); nameserverPublicIPv4Raw != "" {
		nameserverPublicIPv4 = net.ParseIP(nameserverPublicIPv4Raw)
		if nameserverPublicIPv4 == nil {
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

	// Determine subdomain
	subdomain := strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(question.Name), zone), ".")

	// Verify domain existence and determine records
	var records []dns.RR
	code := dns.RcodeSuccess

	if question.Qtype == dns.TypeNS { // NS records are available everywhere in the zone, even for non-existens domains
		records = append(records, &dns.NS{
			Ns: "ns." + zone,
		})
	}

	if len(subdomain) == 0 { // <zone> - this must never be NXDOMAIN
		switch question.Qtype {
		case dns.TypeA:
			for _, websiteIPv4 := range websiteA {
				records = append(records, &dns.A{
					A: websiteIPv4,
				})
			}
		case dns.TypeAAAA:
			for _, websiteIPv6 := range websiteAAAA {
				records = append(records, &dns.AAAA{
					AAAA: websiteIPv6,
				})
			}
		}
	} else if subdomain == "www" { // www.<zone>
		switch question.Qtype {
		case dns.TypeCNAME:
			records = append(records, &dns.CNAME{
				Target: zone,
			})
		case dns.TypeA:
			if len(websiteA) == 0 && len(websiteAAAA) == 0 {
				code = dns.RcodeNameError
				break
			} else if len(websiteA) == 0 {
				break
			}
			// There is a CNAME for www, so the CNAME is returned, with A records for the canonical name attached
			records = append(records, &dns.CNAME{
				Target: zone,
				Hdr: dns.RR_Header{
					Rrtype: dns.TypeCNAME,
				},
			})
			for _, websiteIPv6 := range websiteA {
				records = append(records, &dns.A{
					A: websiteIPv6,
					Hdr: dns.RR_Header{
						Name: "www." + zone,
					},
				})
			}
		case dns.TypeAAAA:
			if len(websiteA) == 0 && len(websiteAAAA) == 0 {
				code = dns.RcodeNameError
				break
			} else if len(websiteAAAA) == 0 {
				break
			}
			// There is a CNAME for www, so the CNAME is returned, with AAAA records for the canonical name attached
			records = append(records, &dns.CNAME{
				Target: zone,
				Hdr: dns.RR_Header{
					Rrtype: dns.TypeCNAME,
				},
			})
			for _, websiteIPv4 := range websiteAAAA {
				records = append(records, &dns.AAAA{
					AAAA: websiteIPv4,
					Hdr: dns.RR_Header{
						Name: "www." + zone,
					},
				})
			}
		}
	} else if subdomain == "ns" { // ns.<zone>
		switch question.Qtype {
		case dns.TypeA:
			records = append(records, &dns.A{
				A: nameserverPublicIPv4,
			})
		}
	} else if subdomainIPv4 := parseIPv4Subdomain(subdomain); subdomainIPv4 != nil { // <ipv4>.<zone>
		switch question.Qtype {
		case dns.TypeA:
			records = append(records, &dns.A{
				A: subdomainIPv4,
			})
		}
	} else if subdomainIPv6 := parseIPv6Subdomain(subdomain); subdomainIPv6 != nil { // <ipv6>.<zone>
		switch question.Qtype {
		case dns.TypeAAAA:
			records = append(records, &dns.AAAA{
				AAAA: subdomainIPv6,
			})
		}
	} else {
		code = dns.RcodeNameError
	}

	return records, code
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
		if header.Name == "" {
			header.Name = question.Name // The RR name can be different than the question's if there was a CNAME
		}
		if header.Rrtype == 0 { // The RR type can be different than the question's if there was a CNAME
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
		}
		header.Class = dns.ClassINET
		header.Ttl = 3600 // TODO: Increase when tests are comphrensive enough
		msg.Answer = append(msg.Answer, answer)
	}
	msg.SetRcode(r, rcode)

	w.WriteMsg(msg)
}
