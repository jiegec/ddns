package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/juju/loggo"
	"github.com/miekg/dns"
	"github.com/urfave/cli"
)

var logger = loggo.GetLogger("ddns")

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
	orig, err := provider.Get(name, record)
	logger.Infof("The '%s' record of %s was %s.\n", record, name, orig)
	for _, r := range orig {
		if r == value {
			// Found
			logger.Infof("No changes made.\n")
			return err
		}
	}
	logger.Infof("Set '%s' record of %s to %s.\n", record, name, value)
	err = provider.Set(name, value, record)
	return err
}

func action(c *cli.Context) {
	loggo.ConfigureLoggers("ddns=INFO")

	homedir, _ := os.UserHomeDir()
	conf := path.Join(homedir, ".ddns")
	err := parseSettingsFile(conf)
	if err != nil {
		logger.Errorf("Failed to parse config: %s", err)
		return
	}

	err = mergeCliSettings(c)
	if err != nil {
		logger.Errorf("Bad settings: %s", err)
		return
	}

	var provider DDNSProvider
	if settings.Provider == "route53" {
		provider, err = newRoute53Provider()
	} else if settings.Provider == "rfc2136" {
		provider, err = newRfc2136Provider()
	} else {
		logger.Errorf("Unsupported ddns provider: %s", settings.Provider)
		return
	}
	if err != nil {
		logger.Errorf("Failed to setup ddns provider: %s", err)
		return
	}

	domain := settings.DomainName
	hostname, err := os.Hostname()

	// use the first part before "."
	parts := strings.Split(hostname, ".")
	hostname = parts[0]

	logger.Infof("Got hostname: %s", hostname)
	if err != nil {
		logger.Errorf("Failed to get hostname")
		return
	}

	name := dns.Fqdn(fmt.Sprintf("%s.%s", hostname, domain))

	ip4, err4 := getIP(false)
	ip6, err6 := getIP(true)
	if err4 == nil {
		err = update(name, ip4, "A", provider)
		if err != nil {
			logger.Errorf("Failed to set dns for %s: %s", name, err)
			return
		}
	}

	if err6 == nil {
		err = update(name, ip6, "AAAA", provider)
		if err != nil {
			logger.Errorf("Failed to set dns for %s: %s", name, err)
			return
		}
	}

	if err4 != nil && err6 != nil {
		logger.Errorf("Failed to get both public ip v4 and v6")
		return
	}

	bmc, err := getBMCIP()
	if err == nil {
		name := dns.Fqdn(fmt.Sprintf("bmc-%s.%s", hostname, domain))
		err = update(name, bmc, "A", provider)
		if err != nil {
			logger.Errorf("Failed to set dns for %s: %s", name, err)
			return
		}
	}

	return
}

func main() {
	app := &cli.App{
		Name:    "ddns",
		Usage:   "DDNS util",
		Version: "1.1",
		Flags: []cli.Flag{
			// global settings
			&cli.StringFlag{
				Name:  "domain, d",
				Usage: "Domain name",
			},
			&cli.StringFlag{
				Name:  "provider, p",
				Usage: "DDNS provider",
			},
			// route53
			&cli.StringFlag{
				Name:  "id, i",
				Usage: "Hosted zone id (for route53)",
			},
			// rfc2136
			&cli.StringFlag{
				Name:  "ns, n",
				Usage: "Nameserver address (for rfc2136)",
			},
			&cli.StringFlag{
				Name:  "algo, a",
				Usage: "TSIG Algorithm (for rfc2136)",
			},
			&cli.StringFlag{
				Name:  "key, k",
				Usage: "TSIG Key (for rfc2136)",
			},
			&cli.StringFlag{
				Name:  "secret, s",
				Usage: "TSIG Secret (for rfc2136)",
			},
		},
		Action: action,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
