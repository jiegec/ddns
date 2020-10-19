package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/miekg/dns"
	"github.com/urfave/cli"
)

func getIP(ipv6 bool) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 2,
	}
	url := "http://api.ipify.org"
	if ipv6 {
		url = "http://api6.ipify.org"
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func setDNS(c *cli.Context, name *string, ip *string, record *string) error {
	id := c.String("id")

	sess := session.Must(session.NewSession())
	client := route53.New(sess)
	_, err := client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &id,
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{{
				Action: aws.String("UPSERT"),
				ResourceRecordSet: &route53.ResourceRecordSet{
					TTL:  aws.Int64(300),
					Type: record,
					Name: name,
					ResourceRecords: []*route53.ResourceRecord{{
						Value: ip,
					}},
				},
			}},
		},
	})
	return err
}

func getBMCOutput() (string, error) {
	// check existence
	out, err := exec.Command("which", "ipmitool").Output()
	if err != nil || len(out) == 0 {
		return "", err
	}

	// only support no password sudo
	out, err = exec.Command("sudo", "-n", "ipmitool", "lan", "print").Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func getBMCIP() (string, error) {
	out, err := getBMCOutput()
	if err != nil {
		return "", err
	}

	regex := *regexp.MustCompile(`IP Address\s+:\s([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
	match := regex.FindAllStringSubmatch(out, -1)
	return match[0][1], nil
}

func update(name string, value string, record string, provider DDNSProvider) error {
	orig, err := provider.Get(name, "A")
	for _, r := range orig {
		if r == value {
			// Found
			log.Printf("The '%s' record of %s is already %s\n", record, name, value)
			return err
		}
	}
	log.Printf("Set '%s' record of %s to %s\n", record, name, value)
	err = provider.Set(name, value, record)
	return err
}

func actionRoute53(c *cli.Context) error {
	provider, err := newRoute53Provider(c)
	if err == nil {
		err = action(c, provider)
	}
	return err
}

func actionRfc2136(c *cli.Context) error {
	provider, err := newRfc2136Provider(c)
	if err == nil {
		err = action(c, provider)
	}
	return err
}

func action(c *cli.Context, provider DDNSProvider) error {
	domain := c.GlobalString("domain")
	hostname, err := os.Hostname()

	// use the first part before "."
	parts := strings.Split(hostname, ".")
	hostname = parts[0]

	log.Println("Got hostname", hostname)
	if err != nil {
		log.Println("Failed to get hostname")
		return err
	}

	name := dns.Fqdn(fmt.Sprintf("%s.%s", hostname, domain))

	ip4, err4 := getIP(false)
	ip6, err6 := getIP(true)
	if err4 == nil {
		err = update(name, ip4, "A", provider)
		if err != nil {
			log.Println("Failed to set dns")
			return err
		}
	}

	if err6 == nil {
		err = update(name, ip6, "AAAA", provider)
		if err != nil {
			log.Println("Failed to set dns")
			return err
		}
	}

	if err4 != nil && err6 != nil {
		log.Println("Failed to get both public ip v4 and v6")
		return err4
	}

	bmc, err := getBMCIP()
	if err == nil {
		name := fmt.Sprintf("bmc-%s.%s", hostname, domain)
		err = update(name, bmc, "A", provider)
		if err != nil {
			log.Println("Failed to set dns")
			return err
		}
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:    "ddns",
		Usage:   "DDNS util",
		Version: "1.0",
		Flags: []cli.Flag{&cli.StringFlag{
			Name:     "domain, d",
			Usage:    "Domain name",
			Required: true,
		}},
		Commands: []cli.Command{
			{
				Name:   "route53",
				Usage:  "Use route53 as ddns backend",
				Flags:  route53Flags,
				Action: actionRoute53,
			},
			{
				Name:   "rfc2136",
				Usage:  "Use rfc2136 as ddns backend",
				Flags:  rfc2136Flags,
				Action: actionRfc2136,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
