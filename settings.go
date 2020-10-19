package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli"
)

type Settings struct {
	// global settings
	Provider   string `json:"provider"`
	DomainName string `json:"domain_name"`

	// route53
	DomainZoneID string `json:"domain_zone_id"`

	// rfc2136
	NameServer    string `json:"name_server"`
	TSIGAlgorithm string `json:"sig_algo"`
	TSIGKey       string `json:"sig_key"`
	TSIGSecret    string `json:"sig_secret"`
}

var settings Settings

func parseSettingsFile(path string) error {
	sf, err := os.Open(path)
	if err != nil {
		logger.Debugf("Read config file \"%s\" failed (may be existence or access problem)\n", path)
		return nil
	}
	logger.Debugf("Read config file \"%s\" succeeded\n", path)
	defer sf.Close()
	bv, _ := ioutil.ReadAll(sf)
	json.Unmarshal(bv, &settings)
	return nil
}

func mergeCliSettings(c *cli.Context) error {
	var merged Settings
	// global settings
	merged.Provider = c.GlobalString("provider")
	if len(merged.Provider) == 0 {
		merged.Provider = settings.Provider
	}
	merged.DomainName = c.GlobalString("domain")
	if len(merged.DomainName) == 0 {
		merged.DomainName = settings.DomainName
	}
	if len(merged.DomainName) == 0 {
		return fmt.Errorf("Domain name can't be null")
	}

	// route53
	merged.DomainZoneID = c.GlobalString("id")
	if len(merged.DomainZoneID) == 0 {
		merged.DomainZoneID = settings.DomainZoneID
	}

	// rfc2136
	merged.NameServer = c.GlobalString("ns")
	if len(merged.NameServer) == 0 {
		merged.NameServer = settings.NameServer
	}
	merged.TSIGAlgorithm = c.GlobalString("algo")
	if len(merged.TSIGAlgorithm) == 0 {
		merged.TSIGAlgorithm = settings.TSIGAlgorithm
	}
	merged.TSIGKey = c.GlobalString("key")
	if len(merged.TSIGKey) == 0 {
		merged.TSIGKey = settings.TSIGKey
	}
	merged.TSIGSecret = c.GlobalString("secret")
	if len(merged.TSIGSecret) == 0 {
		merged.TSIGSecret = settings.TSIGSecret
	}

	settings = merged
	return nil
}
