package main

import (
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type rfc2136Provider struct {
	DomainName    string
	NameServer    string
	TSIGAlgorithm string
	TSIGKey       string
	TSIGSecret    string
}

var rfc2136Flags []cli.Flag

func init() {
	rfc2136Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "ns, n",
			Usage:    "Nameserver address",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "algo, a",
			Usage: "TSIG Algorithm",
		},
		&cli.StringFlag{
			Name:  "key, k",
			Usage: "TSIG Key",
		},
		&cli.StringFlag{
			Name:  "secret, s",
			Usage: "TSIG Secret",
		},
	}
}

func newRfc2136Provider(c *cli.Context) (*rfc2136Provider, error) {
	return &rfc2136Provider{
		DomainName:    c.GlobalString("domain"),
		NameServer:    c.String("ns"),
		TSIGAlgorithm: c.String("algo"),
		TSIGKey:       c.String("key"),
		TSIGSecret:    c.String("secret"),
	}, nil
}

// learned from https://github.com/go-acme/lego/blob/master/providers/dns/rfc2136/rfc2136.go
func (p *rfc2136Provider) Set(name string, value string, record string) error {
	var rrs []dns.RR
	if record == "A" {
		rr := new(dns.A)
		rr.Hdr = dns.RR_Header{Name: dns.Fqdn(name), Rrtype: dns.StringToType[record], Class: dns.ClassINET, Ttl: uint32(300)}
		rr.A = net.ParseIP(value)
		rrs = []dns.RR{rr}
	} else if record == "AAAA" {
		rr := new(dns.AAAA)
		rr.Hdr = dns.RR_Header{Name: dns.Fqdn(name), Rrtype: dns.StringToType[record], Class: dns.ClassINET, Ttl: uint32(300)}
		rr.AAAA = net.ParseIP(value)
		rrs = []dns.RR{rr}
	} else {
		return fmt.Errorf("Unsupported record type: %s", record)
	}

	m := new(dns.Msg)
	m.SetUpdate(dns.Fqdn(p.DomainName))
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
	if reply != nil && reply.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("DNS query failed: server replied: %s", dns.RcodeToString[reply.Rcode])
	}
	for _, r := range reply.Answer {
		if r.Header().Name == name {
			if r.Header().Rrtype == dns.TypeA {
				ret = append(ret, r.(*dns.A).A.String())
			} else if r.Header().Rrtype == dns.TypeAAAA {
				ret = append(ret, r.(*dns.AAAA).AAAA.String())
			}
		}
	}
	return ret, err
}
