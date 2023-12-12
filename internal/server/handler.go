package server

import (
	"log"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

const ttl = 86400

type DNSHandler struct {
	zone        string
	websiteA    []net.IP
	websiteAAAA []net.IP
	nsA         []net.IP
	nsAAAA      []net.IP
	rootTXT     []string
}

func (h *DNSHandler) InitFromEnv() {
	h.zone = strings.ToLower(os.Getenv("ZONE"))

	if h.zone == "" {
		log.Fatal("ZONE environment variable must be set")
	}
	if !strings.HasSuffix(h.zone, ".") {
		h.zone += "."
	}
	if websiteIPv4sRaw := os.Getenv("WEBSITE_A"); websiteIPv4sRaw != "" {
		for _, websiteIPv4Raw := range strings.Split(websiteIPv4sRaw, ",") {
			if websiteIPv4 := net.ParseIP(websiteIPv4Raw); websiteIPv4.To4() != nil {
				h.websiteA = append(h.websiteA, websiteIPv4)
			} else {
				log.Fatalf("WEBSITE_A environment variable is invalid: %s", websiteIPv4Raw)
			}
		}
	}
	if websiteIPv6sRaw := os.Getenv("WEBSITE_AAAA"); websiteIPv6sRaw != "" {
		for _, websiteIPv6Raw := range strings.Split(websiteIPv6sRaw, ",") {
			if websiteIPv6 := net.ParseIP(websiteIPv6Raw); websiteIPv6 != nil {
				h.websiteAAAA = append(h.websiteAAAA, websiteIPv6)
			} else {
				log.Fatalf("WEBSITE_AAAA environment variable is invalid: %s", websiteIPv6Raw)
			}
		}
	}
	if nameserverIPv4sRaw := os.Getenv("NAMESERVER_A"); nameserverIPv4sRaw != "" {
		nameserverIPv4sSplit := strings.Split(nameserverIPv4sRaw, ",")
		if len(nameserverIPv4sSplit) > 2 {
			log.Fatal("NAMESERVER_A environment variable must contain at most two addresses")
		}
		for _, nameserverIPv4Raw := range strings.Split(nameserverIPv4sRaw, ",") {
			if nameserverIPv4 := net.ParseIP(nameserverIPv4Raw); nameserverIPv4.To4() != nil {
				h.nsA = append(h.nsA, nameserverIPv4)
			} else {
				log.Fatalf("NAMESERVER_A environment variable is invalid: %s", nameserverIPv4)
			}
		}
	} else {
		log.Fatal("NAMESERVER_A environment variable must be set")
	}
	if nameserverIPv6sRaw := os.Getenv("NAMESERVER_AAAA"); nameserverIPv6sRaw != "" {
		nameserverIPv6sSplit := strings.Split(nameserverIPv6sRaw, ",")
		if len(nameserverIPv6sSplit) > 2 {
			log.Fatal("NAMESERVER_AAAA environment variable must contain at most two addresses")
		} else if len(nameserverIPv6sSplit) > 0 && len(nameserverIPv6sSplit) != len(h.nsA) {
			log.Fatal("If the NAMESERVER_AAAA environment variable is set, it must contain the same number of addresses as NAMESERVER_A")
		}
		for _, nameserverIPv6Raw := range strings.Split(nameserverIPv6sRaw, ",") {
			if nameserverIPv6 := net.ParseIP(nameserverIPv6Raw); nameserverIPv6 != nil {
				h.nsAAAA = append(h.nsAAAA, nameserverIPv6)
			} else {
				log.Fatalf("NAMESERVER_AAAA environment variable is invalid: %s", nameserverIPv6)
			}
		}
	}
	if rootTXTsRaw := os.Getenv("ROOT_TXT"); rootTXTsRaw != "" {
		h.rootTXT = strings.Split(rootTXTsRaw, ",")
	}
}

// Resolve a question into an answer, an extra record and a response code
func (h *DNSHandler) ResolveRRs(question dns.Question) ([]dns.RR, int) {
	log.Printf("Resolving %s records for %s\n", dns.TypeToString[question.Qtype], question.Name)

	if question.Qclass != dns.ClassINET {
		return nil, dns.RcodeNotImplemented
	}

	// Make sure that the name from the question lies within the zone
	if !strings.HasSuffix(strings.ToLower(question.Name), h.zone) {
		return nil, dns.RcodeNotZone
	}

	// Determine subdomain
	subdomain := strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(question.Name), h.zone), ".")

	// Verify domain existence and determine records
	var records []dns.RR
	code := dns.RcodeSuccess

	if question.Qtype == dns.TypeNS { // NS records are available everywhere in the zone, even for non-existent domains
		records = append(records, &dns.NS{
			Ns: "alpha." + h.zone,
		})
		if len(h.nsA) > 1 {
			records = append(records, &dns.NS{
				Ns: "omega." + h.zone,
			})
		}
	}

	if len(subdomain) == 0 { // <zone> - this must never be NXDOMAIN
		switch question.Qtype {
		case dns.TypeA:
			for _, websiteIPv4 := range h.websiteA {
				records = append(records, &dns.A{
					A: websiteIPv4,
				})
			}
		case dns.TypeAAAA:
			for _, websiteIPv6 := range h.websiteAAAA {
				records = append(records, &dns.AAAA{
					AAAA: websiteIPv6,
				})
			}
		case dns.TypeTXT:
			if len(h.rootTXT) > 0 {
				records = append(records, &dns.TXT{
					Txt: h.rootTXT,
				})
			}
		}
	} else if subdomain == "www" { // www.<zone>
		switch question.Qtype {
		case dns.TypeCNAME:
			if len(h.websiteA) == 0 && len(h.websiteAAAA) == 0 {
				code = dns.RcodeNameError
				break
			}
			records = append(records, &dns.CNAME{
				Target: h.zone,
			})
		case dns.TypeA:
			if len(h.websiteA) == 0 && len(h.websiteAAAA) == 0 {
				code = dns.RcodeNameError
				break
			} else if len(h.websiteA) == 0 {
				break
			}
			// There is a CNAME for www, so the CNAME is returned, with A records for the canonical name attached
			records = append(records, &dns.CNAME{
				Hdr: dns.RR_Header{
					Rrtype: dns.TypeCNAME,
				},
				Target: h.zone,
			})
			for _, websiteIPv6 := range h.websiteA {
				records = append(records, &dns.A{
					Hdr: dns.RR_Header{
						Name: "www." + h.zone,
					},
					A: websiteIPv6,
				})
			}
		case dns.TypeAAAA:
			if len(h.websiteA) == 0 && len(h.websiteAAAA) == 0 {
				code = dns.RcodeNameError
				break
			} else if len(h.websiteAAAA) == 0 {
				break
			}
			// There is a CNAME for www, so the CNAME is returned, with AAAA records for the canonical name attached
			records = append(records, &dns.CNAME{
				Target: h.zone,
				Hdr: dns.RR_Header{
					Rrtype: dns.TypeCNAME,
				},
			})
			for _, websiteIPv4 := range h.websiteAAAA {
				records = append(records, &dns.AAAA{
					Hdr: dns.RR_Header{
						Name: "www." + h.zone,
					},
					AAAA: websiteIPv4,
				})
			}
		}
	} else if subdomain == "alpha" { // alpha.<zone> - first nameserver
		switch question.Qtype {
		case dns.TypeA:
			records = append(records, &dns.A{
				A: h.nsA[0],
			})
		case dns.TypeAAAA:
			if len(h.nsAAAA) > 0 {
				records = append(records, &dns.AAAA{
					AAAA: h.nsAAAA[0],
				})
			}
		}
	} else if subdomain == "omega" { // omega.<zone> - second nameserver
		switch question.Qtype {
		case dns.TypeA:
			if len(h.nsA) > 1 {
				records = append(records, &dns.A{
					A: h.nsA[1],
				})
			} else {
				code = dns.RcodeNameError
			}
		case dns.TypeAAAA:
			if len(h.nsAAAA) > 1 {
				records = append(records, &dns.AAAA{
					AAAA: h.nsAAAA[1],
				})
			} else {
				code = dns.RcodeNameError
			}
		}
	} else if subdomainIPv6 := parseIPv6Subdomain(subdomain); subdomainIPv6 != nil { // <ipv6>.<zone>
		switch question.Qtype {
		case dns.TypeAAAA:
			records = append(records, &dns.AAAA{
				AAAA: subdomainIPv6,
			})
		}
	} else if subdomainIPv4 := parseIPv4Subdomain(subdomain); subdomainIPv4 != nil { // <ipv4>.<zone>
		switch question.Qtype {
		case dns.TypeA:
			records = append(records, &dns.A{
				A: subdomainIPv4,
			})
		}
	} else {
		code = dns.RcodeNameError
	}

	// Make the headers all neat
	for _, answer := range records {
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
			case dns.TypeTXT:
				header.Rrtype = dns.TypeTXT
			}
		}
		header.Class = dns.ClassINET
		header.Ttl = ttl
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
	answers, rcode := h.ResolveRRs(question)
	for _, answer := range answers {
		msg.Answer = append(msg.Answer, answer)
	}
	msg.SetRcode(r, rcode)

	w.WriteMsg(msg)
}
