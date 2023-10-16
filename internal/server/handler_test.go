package server

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

var testNsA = net.ParseIP("127.0.0.1")
var websiteA = net.ParseIP("192.168.0.1")
var websiteAAAA = net.ParseIP("2001:db8::1")

func TestResolvesForNameservers(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// ns.example.com

	answers_ns, rcode_ns := handler.ResolveRRs(dns.Question{
		Name:   "ns.example.com.",
		Qtype:  dns.TypeNS,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_ns)
	assert.Equal(t, []dns.RR{
		&dns.NS{
			Hdr: dns.RR_Header{
				Name:   "ns.example.com.",
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Ns: "ns.example.com.",
		},
	}, answers_ns)

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "ns.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   "ns.example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: testNsA,
		},
	}, answers_a)

	// example.com

	answers_root_ns, rcode_root_ns := handler.ResolveRRs(dns.Question{
		Name:   "example.com.",
		Qtype:  dns.TypeNS,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_root_ns)
	assert.Equal(t, []dns.RR{
		&dns.NS{
			Hdr: dns.RR_Header{
				Name:   "example.com.",
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Ns: "ns.example.com.",
		},
	}, answers_root_ns)

}

func TestDoesNotResolveForWebsiteIfUnconfigured(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// example.com

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR(nil), answers_a)

	answers_cname, rcode_cname := handler.ResolveRRs(dns.Question{
		Name:   "example.com.",
		Qtype:  dns.TypeCNAME,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_cname)
	assert.Equal(t, []dns.RR(nil), answers_cname)

	// www.example.com

	answers_www_a, rcode_www_a := handler.ResolveRRs(dns.Question{
		Name:   "www.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeNameError, rcode_www_a)
	assert.Equal(t, []dns.RR(nil), answers_www_a)

	answers_www_cname, rcode_www_cname := handler.ResolveRRs(dns.Question{
		Name:   "www.example.com.",
		Qtype:  dns.TypeCNAME,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeNameError, rcode_www_cname)
	assert.Equal(t, []dns.RR(nil), answers_www_cname)
}

func TestResolvesForWebsiteIfConfigured(t *testing.T) {
	handler := DNSHandler{
		zone:        "example.com.",
		nsA:         []net.IP{testNsA},
		websiteA:    []net.IP{websiteA},
		websiteAAAA: []net.IP{websiteAAAA},
	}

	// example.com

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   "example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: websiteA,
		},
	}, answers_a)

	answers_cname, rcode_cname := handler.ResolveRRs(dns.Question{
		Name:   "example.com.",
		Qtype:  dns.TypeCNAME,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_cname)
	assert.Equal(t, []dns.RR(nil), answers_cname)

	// www.example.com

	answers_www_a, rcode_www_a := handler.ResolveRRs(dns.Question{
		Name:   "www.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_www_a)
	assert.Equal(t, []dns.RR{
		&dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   "www.example.com.",
				Rrtype: dns.TypeCNAME,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Target: "example.com.",
		},
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   "www.example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: websiteA,
		},
	}, answers_www_a)

	answers_www_cname, rcode_www_cname := handler.ResolveRRs(dns.Question{
		Name:   "www.example.com.",
		Qtype:  dns.TypeCNAME,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_www_cname)
	assert.Equal(t, []dns.RR{
		&dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   "www.example.com.",
				Rrtype: dns.TypeCNAME,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Target: "example.com.",
		},
	}, answers_www_cname)
}

func TestResolvesCorrectIPv4SubdomainWithDots(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// 127.0.0.1.example.com

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "127.0.0.1.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   "127.0.0.1.example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: net.ParseIP("127.0.0.1"),
		},
	}, answers_a)

	answers_aaaa, rcode_aaaa := handler.ResolveRRs(dns.Question{
		Name:   "127.0.0.1.example.com.",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_aaaa)
	assert.Equal(t, []dns.RR(nil), answers_aaaa)
}

func TestResolvesCorrectIPv4SubdomainWithDotsNamed(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// foo.127.0.0.1.example.com

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "foo.127.0.0.1.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   "foo.127.0.0.1.example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: net.ParseIP("127.0.0.1"),
		},
	}, answers_a)

	answers_aaaa, rcode_aaaa := handler.ResolveRRs(dns.Question{
		Name:   "foo.127.0.0.1.example.com.",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_aaaa)
	assert.Equal(t, []dns.RR(nil), answers_aaaa)
}

func TestResolvesCorrectIPv4SubdomainWithDashes(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// 123-0-0-4.example.com

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "200-0-0-4.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   "200-0-0-4.example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: net.ParseIP("200.0.0.4"),
		},
	}, answers_a)

	answers_aaaa, rcode_aaaa := handler.ResolveRRs(dns.Question{
		Name:   "200-0-0-4.example.com.",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_aaaa)
	assert.Equal(t, []dns.RR(nil), answers_aaaa)
}

func TestResolvesCorrectIPv4SubdomainWithDashesNamed(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// foo.123-0-0-4.example.com

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "foo.200-0-0-4.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   "foo.200-0-0-4.example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: net.ParseIP("200.0.0.4"),
		},
	}, answers_a)

	answers_aaaa, rcode_aaaa := handler.ResolveRRs(dns.Question{
		Name:   "foo.200-0-0-4.example.com.",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_aaaa)
	assert.Equal(t, []dns.RR(nil), answers_aaaa)
}

func TestResolvesCorrectIPv6SubdomainWithDots(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// 2001.db8.0.0.0.0.0.1.example.com

	answers_aaaa, rcode_aaaa := handler.ResolveRRs(dns.Question{
		Name:   "2001.db8.0.0.0.0.0.1.example.com.",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_aaaa)
	assert.Equal(t, []dns.RR{
		&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   "2001.db8.0.0.0.0.0.1.example.com.",
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			AAAA: net.ParseIP("2001:db8::1"),
		},
	}, answers_aaaa)

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "2001.db8.0.0.0.0.0.1.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR(nil), answers_a)
}

func TestResolvesCorrectIPv6SubdomainWithDotsNamed(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// foo.2001.db8.0.0.0.0.0.1.example.com

	answers_aaaa, rcode_aaaa := handler.ResolveRRs(dns.Question{
		Name:   "foo.2001.db8.0.0.0.0.0.1.example.com.",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_aaaa)
	assert.Equal(t, []dns.RR{
		&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   "foo.2001.db8.0.0.0.0.0.1.example.com.",
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			AAAA: net.ParseIP("2001:db8::1"),
		},
	}, answers_aaaa)

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "foo.2001.db8.0.0.0.0.0.1.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR(nil), answers_a)
}

func TestResolvesCorrectIPv6SubdomainWithDashes(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// 2001-db8--1.example.com

	answers_aaaa, rcode_aaaa := handler.ResolveRRs(dns.Question{
		Name:   "2001-db8--1.example.com.",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_aaaa)
	assert.Equal(t, []dns.RR{
		&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   "2001-db8--1.example.com.",
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			AAAA: net.ParseIP("2001:db8::1"),
		},
	}, answers_aaaa)

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "2001-db8--1.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR(nil), answers_a)
}

func TestResolvesCorrectIPv6SubdomainWithDashesNamed(t *testing.T) {
	handler := DNSHandler{
		zone: "example.com.",
		nsA:  []net.IP{testNsA},
	}

	// foo.2001-db8--1.example.com

	answers_aaaa, rcode_aaaa := handler.ResolveRRs(dns.Question{
		Name:   "foo.2001-db8--1.example.com.",
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_aaaa)
	assert.Equal(t, []dns.RR{
		&dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   "foo.2001-db8--1.example.com.",
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			AAAA: net.ParseIP("2001:db8::1"),
		},
	}, answers_aaaa)

	answers_a, rcode_a := handler.ResolveRRs(dns.Question{
		Name:   "foo.2001-db8--1.example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	assert.Equal(t, dns.RcodeSuccess, rcode_a)
	assert.Equal(t, []dns.RR(nil), answers_a)
}
