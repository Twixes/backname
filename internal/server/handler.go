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
	siteCname            = strings.TrimPrefix(strings.ToLower(os.Getenv("SITE_CNAME")), ".")
	nameserverPublicIPv4 net.IP
)

func init() {
	if zone == "" {
		log.Fatal("ZONE environment variable must be set")
	}
	if !strings.HasSuffix(zone, ".") {
		zone += "."
	}
	if siteCname != "" && !strings.HasSuffix(siteCname, ".") {
		siteCname += "."
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
func resolve(question dns.Question) (dns.RR, dns.RR, int) {
	log.Printf("Resolving %s records for %s\n", dns.TypeToString[question.Qtype], question.Name)

	// Make sure that the name from the question lies within the zone
	if !strings.HasSuffix(strings.ToLower(question.Name), zone) {
		return nil, nil, dns.RcodeNotZone
	}

	subdomainsString := strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(question.Name), zone), ".")
	var subdomains []string
	if subdomainsString != "" {
		subdomains = strings.Split(subdomainsString, ".")
	}

	// Verify domain existence
	var recordA net.IP
	var recordAAAA net.IP
	var recordCNAME string
	var recordNS string = "ns." + zone
	var recordNSGlueA net.IP = nameserverPublicIPv4 // TODO This may not be needed
	if len(subdomains) == 0 || (len(subdomains) == 1 && subdomains[0] == "www") {
		recordCNAME = siteCname
	} else if len(subdomains) == 1 && subdomains[0] == "ns" {
		recordA = nameserverPublicIPv4
	} else if subdomainIPv4, _ := parseIPv4(subdomains); subdomainIPv4 != nil {
		recordA = subdomainIPv4
	} else if subdomainIPv6, _ := parseIPv6(subdomains); subdomainIPv6 != nil {
		recordAAAA = subdomainIPv6
	} else {
		return nil, nil, dns.RcodeNameError
	}

	if question.Qtype == dns.TypeA {
		if recordA != nil {
			return &dns.A{
				A: recordA,
			}, nil, dns.RcodeSuccess
		}
	} else if question.Qtype == dns.TypeAAAA {
		if recordAAAA != nil {
			return &dns.AAAA{
				AAAA: recordAAAA,
			}, nil, dns.RcodeSuccess
		}
	} else if question.Qtype == dns.TypeCNAME {
		log.Printf("CNAME: %s\n", recordCNAME)
		if recordCNAME != "" {
			return &dns.CNAME{
				Target: siteCname,
			}, nil, dns.RcodeSuccess
		}
	} else if question.Qtype == dns.TypeNS {
		return &dns.NS{
				Ns: recordNS,
			}, &dns.A{ // The glue record
				A: recordNSGlueA,
			}, dns.RcodeSuccess
	}

	return nil, nil, dns.RcodeSuccess
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
	answer, extra, rcode := resolve(question)
	if answer != nil {
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
	if extra != nil {
		header := extra.Header() // Fill in header boilerplate
		header.Class = dns.ClassINET
		header.Name = question.Name
		header.Ttl = 0 // TODO
		header.Rrtype = dns.TypeA
		msg.Extra = append(msg.Extra, extra)
	}
	msg.SetRcode(r, rcode)

	w.WriteMsg(msg)
}
