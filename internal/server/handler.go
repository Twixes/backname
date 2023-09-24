package server

import (
	"log"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

var (
	zone                 = os.Getenv("ZONE")
	zoneParts            = strings.Split(zone, ".")
	nameserverSubdomain  = os.Getenv("NAMESERVER_SUBDOMAIN")
	siteCname            = os.Getenv("SITE_CNAME")
	nameserverPublicIPv4 net.IP
)

func init() {
	if zone == "" {
		log.Fatal("ZONE environment variable must be set")
	}
	if nameserverSubdomain == "" {
		nameserverSubdomain = "ns"
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

func resolve(question dns.Question) (dns.RR, dns.RR, int) {
	log.Printf("Resolving %s records for %s\n", dns.TypeToString[question.Qtype], question.Name)

	nameParts := strings.Split(strings.TrimSuffix(question.Name, "."), ".")

	// Make sure that the name from the question lies within the zone
	if len(nameParts) < len(zoneParts) {
		return nil, nil, dns.RcodeNotZone
	}
	for i := 1; i <= len(zoneParts); i++ {
		if nameParts[len(nameParts)-i] != zoneParts[len(zoneParts)-i] {
			return nil, nil, dns.RcodeNotZone
		}
	}

	if question.Qtype == dns.TypeA {
		subdomains := nameParts[:len(nameParts)-len(zoneParts)]
		var address net.IP
		if len(subdomains) == 1 && subdomains[0] == nameserverSubdomain {
			address = nameserverPublicIPv4
		} else {
			var err error
			address, err = parseIPv4(subdomains)
			if err != nil {
				return nil, nil, dns.RcodeNameError
			}
		}
		return &dns.A{
			Hdr: dns.RR_Header{
				Name:   question.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    0, // TODO
			},
			A: address,
		}, nil, dns.RcodeSuccess
	} else if question.Qtype == dns.TypeAAAA {
		subdomains := nameParts[:len(nameParts)-len(zoneParts)]
		address, err := parseIPv6(subdomains)
		if err != nil {
			return nil, nil, dns.RcodeNameError
		}
		return &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   question.Name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    0, // TODO
			},
			AAAA: address,
		}, nil, dns.RcodeSuccess
	} else if question.Qtype == dns.TypeCNAME {
		subdomains := nameParts[:len(nameParts)-len(zoneParts)]
		if siteCname != "" && len(subdomains) > 0 && !(len(subdomains) == 1 && subdomains[0] == "www") {
			return &dns.CNAME{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeCNAME,
					Class:  dns.ClassINET,
					Ttl:    0, // TODO
				},
				Target: siteCname + "." + zone + ".",
			}, nil, dns.RcodeSuccess
		}
	} else if question.Qtype == dns.TypeNS {
		return &dns.NS{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeNS,
					Class:  dns.ClassINET,
					Ttl:    0, // TODO
				},
				Ns: nameserverSubdomain + "." + zone + ".",
			}, &dns.A{ // The glue record
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    0, // TODO
				},
				A: nameserverPublicIPv4,
			}, dns.RcodeSuccess
	}

	return nil, nil, dns.RcodeServerFailure // TODO: Improve this handling, in particular determine subdomain existence early
}

func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true // TODO: Is this right?

	// Refuse if there are multiple question resource records
	if len(r.Question) != 1 {
		msg.SetRcode(r, dns.RcodeRefused)
		w.WriteMsg(msg)
		return
	}

	question := r.Question[0]
	answer, extra, rcode := resolve(question)
	if answer != nil {
		msg.Answer = append(msg.Answer, answer)
	}
	if extra != nil {
		msg.Extra = append(msg.Extra, extra)
	}
	msg.SetRcode(r, rcode)

	w.WriteMsg(msg)
}
