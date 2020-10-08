package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/urfave/cli"
)

func getIP(ipv6 bool) (string, error) {
	client := &http.Client{}
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

func action(c *cli.Context) error {
	domain := c.String("domain")
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Failed to get hostname")
		return err
	}

	name := fmt.Sprintf("%s.%s", hostname, domain)

	ip4, err4 := getIP(false)
	ip6, err6 := getIP(true)
	if err4 == nil {
		fmt.Printf("Set A record of %s to %s\n", name, ip4)
		err = setDNS(c, &name, &ip4, aws.String("A"))
		if err != nil {
			fmt.Println("Failed to set dns")
			return err
		}
	}

	if err6 == nil {
		fmt.Printf("Set AAAA record of %s to %s\n", name, ip6)
		err = setDNS(c, &name, &ip6, aws.String("AAAA"))
		if err != nil {
			fmt.Println("Failed to set dns")
			return err
		}
	}

	if err4 != nil && err6 != nil {
		fmt.Println("Failed to get both public ip v4 and v6")
		return err4
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:   "ddns",
		Usage:  "DDNS util",
		Action: action,
		Flags: []cli.Flag{&cli.StringFlag{
			Name:     "id, i",
			Usage:    "Hosted zone id",
			Required: true,
		}, &cli.StringFlag{
			Name:     "domain, d",
			Usage:    "Hosted zone domain",
			Required: true,
		}},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
