package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type rfc2136Provider struct {
	NameServer    string
	TSIGAlgorithm string
	TSIGKey       string
	TSIGSecret    string
}

func newRfc2136Provider() (*rfc2136Provider, error) {
	return &rfc2136Provider{
		NameServer:    settings.NameServer,
		TSIGAlgorithm: settings.TSIGAlgorithm,
		TSIGKey:       settings.TSIGKey,
		TSIGSecret:    settings.TSIGSecret,
	}, nil
}

// learned from https://github.com/go-acme/lego/blob/master/providers/dns/rfc2136/rfc2136.go
func (p *rfc2136Provider) Set(name string, value string, record string) error {
	var delRrs []dns.RR
	var rrs []dns.RR
	if record == "A" {
		rr := new(dns.A)
		rr.Hdr = dns.RR_Header{Name: dns.Fqdn(name), Rrtype: dns.StringToType[record], Class: dns.ClassINET, Ttl: uint32(300)}
		rr.A = net.ParseIP(value)
		rrs = []dns.RR{rr}

		delRr := new(dns.A)
		delRr.Hdr = dns.RR_Header{Name: dns.Fqdn(name), Rrtype: dns.StringToType[record], Class: dns.ClassANY, Ttl: uint32(0)}
		delRrs = []dns.RR{delRr}
	} else if record == "AAAA" {
		rr := new(dns.AAAA)
		rr.Hdr = dns.RR_Header{Name: dns.Fqdn(name), Rrtype: dns.StringToType[record], Class: dns.ClassINET, Ttl: uint32(300)}
		rr.AAAA = net.ParseIP(value)
		rrs = []dns.RR{rr}

		delRr := new(dns.AAAA)
		delRr.Hdr = dns.RR_Header{Name: dns.Fqdn(name), Rrtype: dns.StringToType[record], Class: dns.ClassANY, Ttl: uint32(0)}
		delRrs = []dns.RR{delRr}
	} else if record == "PTR" {
		rr := new(dns.PTR)
		rr.Hdr = dns.RR_Header{Name: dns.Fqdn(name), Rrtype: dns.StringToType[record], Class: dns.ClassINET, Ttl: uint32(300)}
		rr.Ptr = value
		rrs = []dns.RR{rr}

		delRr := new(dns.PTR)
		delRr.Hdr = dns.RR_Header{Name: dns.Fqdn(name), Rrtype: dns.StringToType[record], Class: dns.ClassANY, Ttl: uint32(0)}
		delRrs = []dns.RR{delRr}
	} else {
		return fmt.Errorf("Unsupported record type: %s", record)
	}

	parts := strings.Split(dns.Fqdn(name), ".")
	zone := strings.Join(parts[1:], ".")

	m := new(dns.Msg)
	m.SetUpdate(zone)
	m.RemoveRRset(delRrs)
	m.Insert(rrs)

	c := &dns.Client{}
	c.SingleInflight = true

	if len(p.TSIGKey) > 0 && len(p.TSIGSecret) > 0 {
		key := dns.Fqdn(p.TSIGKey)
		alg := dns.Fqdn(p.TSIGAlgorithm)
		m.SetTsig(key, alg, 300, time.Now().Unix())
		c.TsigSecret = map[string]string{key: p.TSIGSecret}
	}

	reply, _, err := c.Exchange(m, p.NameServer)
	if err != nil {
		return errors.Wrap(err, "DNS update failed")
	}
	if reply != nil && reply.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("DNS update failed: server replied: %s", dns.RcodeToString[reply.Rcode])
	}
	return err
}

func (p *rfc2136Provider) Get(name string, record string) ([]string, error) {
	m := new(dns.Msg)
	m.Question = []dns.Question{{Name: dns.Fqdn(name),
		Qtype:  dns.StringToType[record],
		Qclass: dns.ClassINET,
	}}

	c := &dns.Client{}
	c.SingleInflight = true
	reply, _, err := c.Exchange(m, p.NameServer)
	ret := []string{}
	if reply != nil {
		if reply.Rcode != dns.RcodeSuccess {
			return nil, fmt.Errorf("DNS query failed: server replied: %s", dns.RcodeToString[reply.Rcode])
		}
		for _, r := range reply.Answer {
			if r.Header().Name == name {
				if r.Header().Rrtype == dns.TypeA {
					ret = append(ret, r.(*dns.A).A.String())
				} else if r.Header().Rrtype == dns.TypeAAAA {
					ret = append(ret, r.(*dns.AAAA).AAAA.String())
				} else if r.Header().Rrtype == dns.TypePTR {
					ret = append(ret, r.(*dns.PTR).Ptr)
				}
			}
		}
	}
	return ret, err
}
