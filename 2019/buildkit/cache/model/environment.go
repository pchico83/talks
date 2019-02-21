package model

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"bitbucket.org/okteto/okteto/backend/logger"

	"bitbucket.org/okteto/okteto/backend/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
)

const (
	//Demo is the ademo provider
	Demo = "demo"

	//K8 is the kubernetes provider
	K8 = "k8"

	//FixCertificate indicates TLS by copying certificates
	FixCertificate = "fix-certificate"

	//LetsEncrypt indicates TLS by using letsencrypt
	LetsEncrypt = "letsencrypt"
)

//Environment represents a environment.yml file
type Environment struct {
	ID          string       `yaml:"id,omitempty"`
	Name        string       `yaml:"name,omitempty"`
	ProjectName string       `yaml:"project_name,omitempty"`
	DNSProvider *DNSProvider `yaml:"dns,omitempty"`
	Provider    *Provider    `yaml:"provider,omitempty"`
	Registry    *Registry    `yaml:"registry,omitempty"`
}

//DNSProvider represents the info for the cloud provider where the DNS is created
type DNSProvider struct {
	AccessKey    string `yaml:"access_key,omitempty"`
	SecretKey    string `yaml:"secret_key,omitempty"`
	HostedZone   string `yaml:"hosted_zone,omitempty"`
	HostedZoneID string `yaml:"hosted_zone_id,omitempty"`
}

//Provider represents the info for the cloud provider where the service takes place
type Provider struct {
	Type      string             `yaml:"type,omitempty"`
	Username  string             `yaml:"username,omitempty"`
	Password  string             `yaml:"password,omitempty"`
	Endpoint  string             `yaml:"endpoint,omitempty"`
	CaCert    string             `yaml:"ca_cert,omitempty"`
	Ingress   *IngressController `yaml:"ingress,omitempty"`
	InCluster bool               `yaml:"-"`
}

//IngressController represents ingress controller configuration
type IngressController struct {
	AppendProject bool              `yaml:"append_project,omitempty"`
	Domain        string            `yaml:"domain,omitempty"`
	TLS           *TLS              `yaml:"tls,omitempty"`
	Annotations   map[string]string `yaml:"annotations,omitempty"`
}

//TLS represents tls config for ingress
type TLS struct {
	Type        string       `yaml:"type,omitempty"`
	Certificate *Certificate `yaml:"certificate,omitempty"`
}

//Certificate represents certificate config for ingress
type Certificate struct {
	Secret    string `yaml:"secret,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
}

//Registry represents Docker Registry credentials
type Registry struct {
	Server   string `yaml:"server,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

//Validate returns an error for invalid environment.yml files
func (e *Environment) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("'environment.name' is mandatory")
	}
	if !isAlphaNumeric(e.Name) {
		return fmt.Errorf("'environment.name' only allows alphanumeric characters or dashes")
	}
	if e.Provider == nil {
		return fmt.Errorf("'environment.provider' is mandatory")
	}
	if err := e.Provider.Validate(); err != nil {
		return err
	}
	return nil
}

//IsTestProvider return if we are executing on tests
func (p *Provider) IsTestProvider() bool {
	return p.Type == Demo && flag.Lookup("test.v") != nil
}

//IsFreeTierProvider return if we are executing on free tier
func (p *Provider) IsFreeTierProvider() bool {
	return p.Type == Demo && flag.Lookup("test.v") == nil
}

//IsTLSIngress return if we are executing a provider with tls ingress
func (p *Provider) IsTLSIngress() bool {
	if p.Ingress == nil {
		return false
	}
	if p.Ingress.TLS == nil {
		return false
	}
	return true
}

//IsIngress return if we are executing a provider with ingress
func (p *Provider) IsIngress() bool {
	if p.Ingress == nil {
		return false
	}
	if p.Ingress.Domain == "" {
		return false
	}
	return true
}

//LoadDefaultCluster loads the free  tier configuration cluster
func (p *Provider) LoadDefaultCluster() {
	if !p.IsFreeTierProvider() {
		return
	}

	if config.UseInClusterConfig() {
		p.InCluster = true
		p.Endpoint, p.CaCert = config.GetInClusterConfiguration()

		if config.InClusterIngressEnabled() {
			p.Ingress = &IngressController{}
			domain, tlsType, namespace, secret := config.GetInClusterIngressConfiguration()
			p.Ingress.Domain = domain
			p.Ingress.Annotations = map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			}

			if tlsType == LetsEncrypt || tlsType == FixCertificate {
				p.Ingress.TLS = &TLS{
					Type: tlsType,
					Certificate: &Certificate{
						Namespace: namespace,
						Secret:    secret,
					},
				}
			}
		}

		return
	}

	cert, err := ioutil.ReadFile("./certs/free-tier-ca.cert")
	if err != nil {
		logger.Error(errors.Wrap(err, "couldn't read the free-tier cert"))
		return
	}

	p.Username = os.Getenv("OKTETO_FREE_TIER_USERNAME")
	p.Password = os.Getenv("OKTETO_FREE_TIER_PASSWORD")
	p.Endpoint = os.Getenv("OKTETO_FREE_TIER_ENDPOINT")
	p.CaCert = string(cert)

	hostedZone, _, _ := config.GetDNSCredentials()
	hostedZone = strings.TrimSuffix(hostedZone, ".")
	p.Ingress = &IngressController{
		AppendProject: true,
		Domain:        fmt.Sprintf("free-tier.%s", hostedZone),
		TLS: &TLS{
			Type: FixCertificate,
			Certificate: &Certificate{
				Secret:    config.GetCertificateSecret(),
				Namespace: config.GetCertificateNamespace(),
			},
		},
		Annotations: map[string]string{
			"kubernetes.io/ingress.class": "nginx",
		},
	}
}

//Validate returns an error for invalid providers
func (p *Provider) Validate() error {
	switch p.Type {
	case Demo:
		return nil
	case K8:
		if p.Username == "" {
			return fmt.Errorf("'provider.username' cannot be empty")
		}
		if p.Password == "" {
			return fmt.Errorf("'provider.password' cannot be empty")
		}
		if p.Endpoint == "" {
			return fmt.Errorf("'provider.endpoint' cannot be empty")
		}
		if p.CaCert == "" {
			return fmt.Errorf("'provider.ca_cert' cannot be empty")
		}
		if p.Ingress == nil {
			return nil
		}
		if p.Ingress.Domain == "" {
			return fmt.Errorf("'provider.ingress.domain' cannot be empty")
		}
		if p.Ingress.TLS == nil {
			return nil
		}
		if p.Ingress.TLS.Type != LetsEncrypt && p.Ingress.TLS.Type != FixCertificate {
			return fmt.Errorf("'provider.ingress.tls.type' supported types: '%s' and '%s'", FixCertificate, LetsEncrypt)
		}
		if p.Ingress.TLS.Type == FixCertificate {
			if p.Ingress.TLS.Certificate == nil {
				return fmt.Errorf("'provider.ingress.tls.certificate' cannot be empty")
			}
			if p.Ingress.TLS.Certificate.Secret == "" {
				return fmt.Errorf("'provider.ingress.tls.certificate.secret' cannot be empty")
			}
			if p.Ingress.TLS.Certificate.Namespace == "" {
				return fmt.Errorf("'provider.ingress.tls.certificate.namespace' cannot be empty")
			}
		}
		return nil
	default:
		return fmt.Errorf("'provider.type' '%s' is not supported", p.Type)
	}
}

//GetConfig returns a config aws object
func (p *DNSProvider) GetConfig() *aws.Config {
	awsConfig := &aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials(p.AccessKey, p.SecretKey, ""),
	}
	return awsConfig
}

//Validate returns an error for invalid providers
func (p *DNSProvider) Validate() error {
	if p.AccessKey == "" {
		return fmt.Errorf("'provider.access_key' cannot be empty")
	}
	if p.SecretKey == "" {
		return fmt.Errorf("'provider.secret_key' cannot be empty")
	}
	if p.HostedZone == "" {
		return fmt.Errorf("'provider.hosted_zone' cannot be empty")
	}
	svc := route53.New(session.New(), p.GetConfig())
	hostedZonesInput := &route53.ListHostedZonesByNameInput{
		DNSName:  aws.String(p.HostedZone),
		MaxItems: aws.String("1"),
	}
	resp, err := svc.ListHostedZonesByName(hostedZonesInput)
	if err != nil {
		return err
	}
	if len(resp.HostedZones) != 1 {
		return fmt.Errorf("Hosted zone '%s' not found", p.HostedZone)
	}
	p.HostedZoneID = *resp.HostedZones[0].Id
	return nil
}

var dockerConfigTemplate = `
{
	"auths": {
		"%s": {
			"auth": "%s"
		}
	}
}
`

//B64DockerConfig conputes the base64 format of docker credentials
func (e *Environment) B64DockerConfig() string {
	if e.Registry == nil || e.Registry.Username == "" || e.Registry.Password == "" {
		return ""
	}
	if e.Registry.Server == "" {
		e.Registry.Server = "https://index.docker.io/v1/"
	}
	auth := fmt.Sprintf("%s:%s", e.Registry.Username, e.Registry.Password)
	authEncoded := base64.StdEncoding.EncodeToString([]byte(auth))
	config := fmt.Sprintf(dockerConfigTemplate, e.Registry.Server, authEncoded)
	return base64.StdEncoding.EncodeToString([]byte(config))
}
