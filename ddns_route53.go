package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

type Route53Provider struct {
	DomainZoneID string
}

func newRoute53Provider() (*Route53Provider, error) {
	return &Route53Provider{
		DomainZoneID: settings.DomainZoneID,
	}, nil
}

func (p *Route53Provider) Set(name string, value string, record string) error {
	sess := session.Must(session.NewSession())
	client := route53.New(sess)
	_, err := client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &p.DomainZoneID,
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{{
				Action: aws.String("UPSERT"),
				ResourceRecordSet: &route53.ResourceRecordSet{
					TTL:  aws.Int64(300),
					Type: &record,
					Name: &name,
					ResourceRecords: []*route53.ResourceRecord{{
						Value: &value,
					}},
				},
			}},
		},
	})
	return err
}

func (p *Route53Provider) Get(name string, record string) ([]string, error) {
	sess := session.Must(session.NewSession())
	client := route53.New(sess)
	resp, err := client.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId:    &p.DomainZoneID,
		StartRecordName: &name,
		StartRecordType: &record,
	})
	ret := []string{}
	// filter queried fields
	for _, entry := range resp.ResourceRecordSets {
		if *entry.Name == name && *entry.Type == record {
			for _, record := range entry.ResourceRecords {
				ret = append(ret, *record.Value)
			}
		}
	}
	return ret, err
}
