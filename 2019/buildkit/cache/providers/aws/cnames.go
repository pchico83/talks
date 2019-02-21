package aws

import (
	"fmt"
	logger "log"
	"net"
	"time"

	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

//Create creates a CNAME resolving to a service
func Create(s *model.Service, e *model.Environment, target, targetType string, log *logger.Logger) error {
	recordName := fmt.Sprintf("%s.", s.GetDNS(e))
	log.Printf("Waiting for %s to resolve to %s...", recordName, target)
	if err := update(s, e, target, targetType, "UPSERT"); err != nil {
		return err
	}
	tries := 0
	for tries < 20 {
		addrs, err := net.LookupHost(target)
		if err == nil && len(addrs) != 0 {
			return nil
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("DNS not working after 3 minutes")
}

//Destroy destroys a CNAME resolving to a service
func Destroy(s *model.Service, e *model.Environment, target, targetType string) error {
	return update(s, e, target, targetType, "DELETE")
}

func update(s *model.Service, e *model.Environment, target, targetType, action string) error {
	svc := route53.New(session.New(), e.DNSProvider.GetConfig())
	recordName := fmt.Sprintf("%s.", s.GetDNS(e))
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String(action),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(recordName),
						Type: aws.String(targetType),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(target),
							},
						},
						TTL: aws.Int64(60),
					},
				},
			},
		},
		HostedZoneId: aws.String(e.DNSProvider.HostedZoneID),
	}
	_, err := svc.ChangeResourceRecordSets(params)
	return err
}
