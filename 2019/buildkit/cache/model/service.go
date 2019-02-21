package model

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var isAlphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9]*$`).MatchString

// ServiceStatus is the current state of a service
type ServiceStatus string

const (
	//CreatingService is the state when the service is being created
	CreatingService ServiceStatus = "creating"

	//CreatedService is the state when the service is created
	CreatedService ServiceStatus = "created"

	//DeployingService is the state when the service is being deployed
	DeployingService ServiceStatus = "deploying"

	//DeployedService is the state when the service is deployed
	DeployedService ServiceStatus = "deployed"

	//DevDeployingService is the state when the service is being deployed
	DevDeployingService ServiceStatus = "devdeploying"

	//DevDeployedService is the state when the service is deployed
	DevDeployedService ServiceStatus = "devdeployed"

	//DestroyingService is the state when the service is being destroyed
	DestroyingService ServiceStatus = "destroying"

	//DestroyedService is the state when the service is destroyed its Service
	DestroyedService ServiceStatus = "destroyed"

	//FailedService is the state when the service failed to change state
	FailedService ServiceStatus = "failed"

	//Unknown is the state when the service is in an unknown state
	Unknown ServiceStatus = "unknown"

	//ProjectName for resolving the service to project name
	ProjectName = "$PROJECT_NAME"
)

// ServiceLinks contains links to the activities and to itself
type ServiceLinks struct {
	Activities string `json:"activities,omitempty"`
	Self       string `json:"self,omitempty"`
}

//Service represents a service.yml file
type Service struct {
	Model
	Name         string        `json:"name" yaml:"name,omitempty"`
	Status       ServiceStatus `json:"state" gorm:"index"  yaml:"-"`
	ProjectID    string        `json:"project,omitempty" yaml:"-" gorm:"index"`
	Dev          bool          `json:"dev" yaml:"-"`
	CreatedBy    string        `json:"-" yaml:"-" gorm:"index"`
	IsDemo       bool          `json:"-" yaml:"-" gorm:"index"`
	DNS          string        `json:"-" gorm:"dns" yaml:"-"`
	Manifest     string        `json:"manifest,omitempty" gorm:"manifest"  yaml:"-"`
	GHRepoLinkID string        `json:"-,omitempty" yaml:"-" gorm:"index"`

	// YAML content
	Replicas    int                   `json:"replicas,omitempty" yaml:"replicas,omitempty" gorm:"-"`
	GracePeriod int                   `json:"grace_period,omitempty" yaml:"grace_period,omitempty" gorm:"-"`
	Containers  map[string]*Container `json:"containers,omitempty" yaml:"containers,omitempty" gorm:"-"`
	Volumes     map[string]*Volume    `json:"volumes,omitempty" yaml:"volumes,omitempty" gorm:"-"`
	Labels      map[string]string     `json:"labels,omitempty" yaml:"labels,omitempty" gorm:"-"`

	// Linked resources
	Activities []Activity `json:"activities,omitempty" yaml:"-"`

	Endpoints []string     `json:"endpoints,omitempty" gorm:"-"  yaml:"-"`
	Links     ServiceLinks `json:"links,omitempty" gorm:"-"  yaml:"-"`
}

//Container represents a container in a service.yml file
type Container struct {
	Image       string            `json:"image,omitempty" yaml:"image,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`
	Command     string            `json:"command,omitempty" yaml:"command,omitempty"`
	Arguments   []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Mounts      map[string]*Mount `json:"mounts,omitempty" yaml:"mounts,omitempty" gorm:"-"`
	Ports       []string          `json:"ports,omitempty" yaml:"ports,omitempty"`
	Ingress     []*Ingress        `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	Expose      []string          `json:"expose,omitempty" yaml:"expose,omitempty"`
	Environment []*EnvVar         `json:"environment,omitempty" yaml:"environment,omitempty"`
	Resources   *Resources        `json:"resources,omitempty" yaml:"resources,omitempty"`
	Development *Development      `json:"dev,omitempty" yaml:"dev,omitempty"`
}

//Volume represents a volume in a service.yml file
type Volume struct {
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	Persistent bool   `json:"persistent,omitempty" yaml:"persistent,omitempty"`
	Size       string `json:"size,omitempty" yaml:"size,omitempty"`
}

//Mount represents a volume mount in a service.yml file
type Mount struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

//Ingress represents an ingress rule
type Ingress struct {
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	Port string `json:"port,omitempty" yaml:"port,omitempty"`
}

//EnvVar represents a container envvar
type EnvVar struct {
	Name  string
	Value string
}

//Resources represents the container resources
type Resources struct {
	Limits   *Resource `json:"limits,omitempty" yaml:"limits,omitempty"`
	Requests *Resource `json:"requests,omitempty" yaml:"requests,omitempty"`
}

//Resource represents a container resource
type Resource struct {
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`
	CPU    string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
}

//UnmarshalYAML sets the default value of replica to 1
func (s *Service) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawService Service
	raw := rawService{}
	raw.Replicas = 1
	raw.GracePeriod = 30

	if err := unmarshal(&raw); err != nil {
		return err
	}

	*s = Service(raw)

	s.translatePorts()
	s.translateVolumes()
	return nil
}

func contains(array map[string]*Volume, elem string) bool {
	for _, a := range array {
		if a.Name == elem {
			return true
		}
	}
	return false
}

//Validate returns an error for invalid service.yml files
func (s *Service) Validate() *AppError {
	if s.Name == "" {
		return &AppError{Status: http.StatusBadRequest, Code: MissingName, Message: "'service.name' is mandatory"}
	}
	if !isAlphaNumeric(s.Name) {
		return &AppError{Status: http.StatusBadRequest, Code: InvalidName, Message: "'service.name' only allows alphanumeric characters or dashes"}
	}

	if s.GracePeriod < 0 {
		return &AppError{Status: http.StatusBadRequest, Code: InvalidGracePeriod, Message: "'service.grace_period' must be greater than zero or zero for no grace period"}
	}

	if s.Replicas < 1 {
		return &AppError{Status: http.StatusBadRequest, Code: InvalidReplicaCount, Message: "'service.replicas' must be greater than zero"}
	}
	if s.IsPersistent() && s.Replicas > 1 {
		return &AppError{Status: http.StatusBadRequest, Code: InvalidPersistentReplica, Message: "persistent volumes can only be used with a single replica"}
	}

	if s.Containers == nil || len(s.Containers) == 0 {
		return &AppError{Status: 400, Code: InvalidContainerCount}
	}

	for name, c := range s.Containers {
		if c == nil || c.Image == "" {
			return &AppError{
				Status:  400,
				Code:    MissingContainerImage,
				Data:    map[string]string{"container": name},
				Message: fmt.Sprintf("%s must have an image defined", name),
			}
		}
	}

	devContainerCount := 0
	for nC, c := range s.Containers {
		if c.Development != nil {
			devContainerCount++
		}

		for nV := range c.Mounts {
			if !contains(s.Volumes, nV) {
				return &AppError{
					Status:  http.StatusBadRequest,
					Code:    VolumeNotDefined,
					Data:    map[string]string{"volume": nV, "container": nC},
					Message: fmt.Sprintf("Volume '%s' in container '%s' not defined", nV, nC)}
			}
		}
	}
	if devContainerCount > 1 {
		return &AppError{Status: http.StatusBadRequest, Code: InvalidDevContainerCount, Message: "Services can only have one container configured for development"}

	}

	return nil
}

//GetDNS returns the service dns (record name)
func (s *Service) GetDNS(e *Environment) string {
	hostedZone := strings.TrimSuffix(e.DNSProvider.HostedZone, ".")
	return fmt.Sprintf("%s.%s.%s", s.Name, e.Name, hostedZone)
}

//GetLoadBalancerPorts returns the list of load balancer ports of a service
func (s *Service) GetLoadBalancerPorts() []string {
	result := []string{}
	seen := map[string]bool{}
	for _, container := range s.Containers {
		for _, port := range container.Ports {
			if _, ok := seen[port]; !ok {
				result = append(result, port)
				seen[port] = true
			}
		}
	}
	return result
}

//GetIngressRules returns the list of ingress rules of a service
func (s *Service) GetIngressRules(all bool) []*Ingress {
	result := []*Ingress{}
	seen := map[string]bool{}
	for _, container := range s.Containers {
		if all {
			for _, p := range container.Ports {
				i := &Ingress{Host: s.Name, Path: "/", Port: p}
				value := fmt.Sprintf("%s-%s-%s", i.Host, i.Path, i.Port)
				if _, ok := seen[value]; !ok {
					seen[value] = true
					result = append(result, i)
				}
			}
		}
		for _, i := range container.Ingress {
			value := fmt.Sprintf("%s-%s-%s", i.Host, i.Path, i.Port)
			if _, ok := seen[value]; !ok {
				seen[value] = true
				result = append(result, i)
			}
		}
	}
	return result
}

//GetPrivatePorts returns the list of expose ports of a service
func (s *Service) GetPrivatePorts() []string {
	result := []string{}
	seen := map[string]bool{}
	for _, container := range s.Containers {
		for _, p := range container.Ports {
			if _, ok := seen[p]; !ok {
				seen[p] = true
				result = append(result, p)
			}
		}
		for _, i := range container.Ingress {
			if _, ok := seen[i.Port]; !ok {
				seen[i.Port] = true
				result = append(result, i.Port)
			}
		}
		for _, e := range container.Expose {
			if _, ok := seen[e]; !ok {
				seen[e] = true
				result = append(result, e)
			}
		}
	}
	return result
}

//GetIngressHostname returns the ingress hostname of an ingress
func (i *Ingress) GetIngressHostname(e *Environment) string {
	host := i.Host
	if host == ProjectName {
		host = e.ProjectName
	}
	if e.Provider.Ingress.AppendProject {
		host = fmt.Sprintf("%s-dot-%s", host, e.Name)
	}
	return fmt.Sprintf("%s.%s", host, e.Provider.Ingress.Domain)
}

//GetIngressHostnames returns the list of ingress hostnames of a service
func (s *Service) GetIngressHostnames(e *Environment) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, ir := range s.GetIngressRules(true) {
		host := ir.GetIngressHostname(e)
		if _, ok := seen[host]; !ok {
			result = append(result, host)
		}
		seen[host] = true
	}
	return result
}

//GetIngressCertificateName returns the list of ingress hostnames of a service
func (s *Service) GetIngressCertificateName(e *Environment) string {
	switch e.Provider.Ingress.TLS.Type {
	case FixCertificate:
		return e.Provider.Ingress.TLS.Certificate.Secret
	case LetsEncrypt:
		return fmt.Sprintf("%s-%s", s.Name, LetsEncrypt)
	}
	return ""
}

//IsPersistent returns if the service has at least a persistent volume
func (s *Service) IsPersistent() bool {
	for _, v := range s.Volumes {
		if v.Persistent {
			return true
		}
	}
	return false
}

// CalculateStatus calculates the Service state of d based on a
func (s *Service) CalculateStatus(a *Activity) ServiceStatus {
	if a == nil {
		return FailedService
	}

	if a.Status == Failed {
		return FailedService
	}

	switch a.Type {
	case Created:
		if a.Status == InProgress {
			return CreatingService
		}
		return CreatedService

	case Deployed:
		if a.Status == InProgress {
			return DeployingService
		}
		return DeployedService
	case Destroyed:
		if a.Status == InProgress {
			return DestroyingService
		}
		return DestroyedService
	case DevDeployed:
		if a.Status == InProgress {
			return DevDeployingService
		}
		return DevDeployedService
	default:
		return Unknown
	}
}

// CanDeploy returns true if d is in a state that allows a deploy operation
func (s *Service) CanDeploy() bool {
	if s.Status == CreatingService || s.Status == DeployingService || s.Status == DestroyingService ||
		s.Status == DevDeployingService || s.Status == DestroyedService {
		return false
	}

	return true
}

// CanDestroy returns true if d is in a state that allows a destroy operation
func (s *Service) CanDestroy() bool {
	if s.Status == CreatingService || s.Status == DeployingService || s.Status == DevDeployingService ||
		s.Status == DestroyingService || s.Status == DestroyedService {
		return false
	}

	return true
}

// CanEnableDev returns true if the service is in an state that allows enabling dev mode
func (s *Service) CanEnableDev() bool {
	return s.Status == CreatedService || s.Status == DeployedService || s.Status == FailedService
}

// IsDestroyed returns true if d is in a state of destruction
func (s *Service) IsDestroyed() bool {
	if s.Status == DestroyingService || s.Status == DestroyedService {
		return true
	}

	return false
}

//FullName returns the full name if a volume
func (v *Volume) FullName(s *Service, e *Environment) string {
	return fmt.Sprintf("%s.%s.%s", v.Name, s.Name, e.Name)
}
