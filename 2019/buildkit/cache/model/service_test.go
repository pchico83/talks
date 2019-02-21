package model

import (
	"io/ioutil"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestValidateService(t *testing.T) {
	tests := []struct {
		name        string
		s           *Service
		expectError bool
	}{
		{
			name:        "basic",
			expectError: false,
			s: &Service{
				Replicas: 1,
				Name:     "test",
				Containers: map[string]*Container{
					"welcome": &Container{
						Image: "okteto/welcome",
					},
				},
			},
		},
		{
			name:        "dev",
			expectError: false,
			s: &Service{
				Name:     "test",
				Replicas: 1,
				Containers: map[string]*Container{
					"dev": &Container{
						Image: "ubuntu",
						Development: &Development{
							Image: "dev",
						},
					},
					"nginx": &Container{
						Image: "nginx",
					},
				},
			},
		},
		{
			name:        "multi-dev",
			expectError: true,
			s: &Service{
				Name:     "test",
				Replicas: 1,
				Containers: map[string]*Container{
					"dev": &Container{
						Image: "ubuntu",
						Development: &Development{
							Image: "dev",
						},
					},
					"nginx": &Container{
						Image: "nginx",
						Development: &Development{
							Image: "dev",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("did't got the expected error")
				}

			} else {
				if err != nil {
					t.Errorf("got uexpected error: %s", err.Error())
				}
			}
		})
	}
}

func TestGetDNS(t *testing.T) {
	s := Service{Name: "test"}
	e := Environment{
		Name: "env",
		DNSProvider: &DNSProvider{
			HostedZone: "example.com",
		},
	}
	dns := s.GetDNS(&e)
	if dns != "test.env.example.com" {
		t.Errorf("Expected: test.env.example.com \n Received: %s", dns)
	}
}

func TestGetLoadBalancerPorts(t *testing.T) {
	s := Service{
		Name: "test",
		Containers: map[string]*Container{
			"nginx": &Container{
				Ports: []string{"80", "443"},
				Ingress: []*Ingress{
					&Ingress{
						Host: "api",
						Path: "/",
						Port: "5000",
					},
					&Ingress{
						Host: "api",
						Path: "/v2",
						Port: "5000",
					},
					&Ingress{
						Host: "api",
						Path: "/v1",
						Port: "5001",
					},
				},
				Expose: []string{"80", "4443"},
			},
			"api": &Container{
				Ports:  []string{"80", "8080"},
				Expose: []string{"80", "8081"},
			},
		},
	}
	ports := s.GetLoadBalancerPorts()
	sort.Strings(ports)
	if !reflect.DeepEqual(ports, []string{"443", "80", "8080"}) {
		t.Errorf("Expected: 443, 80, 8080 \n Received: %+v", ports)
	}
}

func TestGetIngressRules(t *testing.T) {
	s := Service{
		Name: "test",
		Containers: map[string]*Container{
			"nginx": &Container{
				Ports: []string{"80"},
				Ingress: []*Ingress{
					&Ingress{
						Host: "api",
						Path: "/v1",
						Port: "80",
					},
					&Ingress{
						Host: "api",
						Path: "/v1",
						Port: "80",
					},
					&Ingress{
						Host: "api",
						Path: "/v1",
						Port: "8080",
					},
				},
				Expose: []string{"80", "4443"},
			},
		},
	}
	irs := s.GetIngressRules(true)
	expected := []*Ingress{
		&Ingress{
			Host: "test",
			Path: "/",
			Port: "80",
		},
		&Ingress{
			Host: "api",
			Path: "/v1",
			Port: "80",
		},
		&Ingress{
			Host: "api",
			Path: "/v1",
			Port: "8080",
		},
	}
	if !reflect.DeepEqual(irs, expected) {
		t.Errorf("Expected: %+v \n Received: %+v", expected, irs)
	}
}

func TestGetPrivatePorts(t *testing.T) {
	s := Service{
		Name: "test",
		Containers: map[string]*Container{
			"nginx": &Container{
				Ports: []string{"80", "443"},
				Ingress: []*Ingress{
					&Ingress{
						Host: "api",
						Path: "/",
						Port: "5000",
					},
					&Ingress{
						Host: "api",
						Path: "/v2",
						Port: "5000",
					},
					&Ingress{
						Host: "api",
						Path: "/v1",
						Port: "5001",
					},
				},
				Expose: []string{"80", "4443"},
			},
			"api": &Container{
				Ports:  []string{"80", "8080"},
				Expose: []string{"80", "8081"},
			},
		},
	}
	ports := s.GetPrivatePorts()
	sort.Strings(ports)
	if !reflect.DeepEqual(ports, []string{"443", "4443", "5000", "5001", "80", "8080", "8081"}) {
		t.Errorf("Expected: 443, 4443, 5000, 5001, 80, 8080, 8081 \n Received: %+v", ports)
	}
}

func TestGetIngressHostname(t *testing.T) {
	var tests = []struct {
		ingress     Ingress
		environment Environment
		expected    string
	}{
		{
			ingress: Ingress{Host: "api"},
			environment: Environment{
				Name:        "env-1",
				ProjectName: "env",
				Provider: &Provider{
					Ingress: &IngressController{
						Domain: "example.com",
					},
				},
			},
			expected: "api.example.com",
		},
		{
			ingress: Ingress{Host: "api"},
			environment: Environment{
				Name:        "env-1",
				ProjectName: "env",
				Provider: &Provider{
					Ingress: &IngressController{
						AppendProject: true,
						Domain:        "example.com",
					},
				},
			},
			expected: "api-dot-env-1.example.com",
		},
		{
			ingress: Ingress{Host: ProjectName},
			environment: Environment{
				Name:        "env-1",
				ProjectName: "env",
				Provider: &Provider{
					Ingress: &IngressController{
						Domain: "example.com",
					},
				},
			},
			expected: "env.example.com",
		},
	}
	for _, tt := range tests {
		hostname := tt.ingress.GetIngressHostname(&tt.environment)
		if hostname != tt.expected {
			t.Errorf("Expected: %s \n Received: %s", tt.expected, hostname)
		}
	}
}

func TestGetIngressHostnames(t *testing.T) {
	s := Service{
		Name: "test",
		Containers: map[string]*Container{
			"nginx": &Container{
				Ports: []string{"80"},
				Ingress: []*Ingress{
					&Ingress{
						Host: "api",
						Path: "/v1",
						Port: "80",
					},
					&Ingress{
						Host: "api",
						Path: "/v1",
						Port: "8080",
					},
				},
				Expose: []string{"80", "4443"},
			},
		},
	}
	e := Environment{
		Name:        "env-1",
		ProjectName: "env",
		Provider: &Provider{
			Ingress: &IngressController{
				Domain: "example.com",
			},
		},
	}
	hostnames := s.GetIngressHostnames(&e)
	expected := []string{"test.example.com", "api.example.com"}
	if !reflect.DeepEqual(hostnames, expected) {
		t.Errorf("Expected: %+v \n Received: %+v", expected, hostnames)
	}
}

func TestReadResources(t *testing.T) {
	readBytes, err := ioutil.ReadFile("./examples/service_with_resources.yml")
	require.NoError(t, err)
	var s Service
	err = yaml.Unmarshal(readBytes, &s)
	require.NoError(t, err)

	if s.GracePeriod != 90 {
		t.Fatal("didn't parse grace period")
	}

	container := s.Containers["nginx"]
	if container.Resources == nil {
		t.Fatal("Container.Resources is nil")
	}
	if container.Resources.Limits == nil {
		t.Fatal("Container.Resources.Limits is nil")
	}
	if container.Resources.Limits.Memory != "128Mi" {
		t.Fatalf("Container.Resources.Limits.Memory is not 128Mi, it is %s", container.Resources.Limits.Memory)
	}
}

func TestEmptyResources(t *testing.T) {
	readBytes, err := ioutil.ReadFile("./examples/service-nginx.yml")
	require.NoError(t, err)
	var s Service
	err = yaml.Unmarshal(readBytes, &s)
	require.NoError(t, err)
	container := s.Containers["nginx"]
	if container.Resources != nil {
		t.Fatal("Container.Resources is not nil")
	}
}

func TestCanDeploy(t *testing.T) {
	var tables = []struct {
		service   Service
		canDeploy bool
	}{
		{Service{Status: CreatedService}, true},
		{Service{Status: DeployedService}, true},
		{Service{Status: CreatingService}, false},
		{Service{Status: DestroyingService}, false},
		{Service{Status: DestroyedService}, false},
		{Service{Status: FailedService}, true},
	}

	for _, tt := range tables {
		if tt.service.CanDeploy() != tt.canDeploy {
			t.Errorf("%s was not %t", tt.service.Status, tt.canDeploy)
		}
	}
}

func TestCanDelete(t *testing.T) {
	var tables = []struct {
		service    Service
		CanDestroy bool
	}{
		{Service{Status: CreatedService}, true},
		{Service{Status: DeployedService}, true},
		{Service{Status: CreatingService}, false},
		{Service{Status: DestroyingService}, false},
		{Service{Status: DestroyedService}, false},
		{Service{Status: FailedService}, true},
	}

	for _, tt := range tables {
		if tt.service.CanDestroy() != tt.CanDestroy {
			t.Errorf("%s was not %t", tt.service.Status, tt.CanDestroy)
		}
	}
}

func TestCalculateState(t *testing.T) {
	var tables = []struct {
		activity Activity
		expected ServiceStatus
	}{
		{Activity{Type: Created, Status: InProgress}, CreatingService},
		{Activity{Type: Created, Status: Completed}, CreatedService},
		{Activity{Type: Created, Status: Failed}, FailedService},
		{Activity{Type: Deployed, Status: InProgress}, DeployingService},
		{Activity{Type: Deployed, Status: Completed}, DeployedService},
		{Activity{Type: Deployed, Status: Failed}, FailedService},
		{Activity{Type: Destroyed, Status: InProgress}, DestroyingService},
		{Activity{Type: Destroyed, Status: Completed}, DestroyedService},
		{Activity{Type: Destroyed, Status: Failed}, FailedService},
	}

	for _, tt := range tables {
		d := Service{}
		if d.CalculateStatus(&tt.activity) != tt.expected {
			t.Errorf("%+v was not %s", tt.activity, tt.expected)
		}
	}
}

func TestServiceIsDestroyed(t *testing.T) {
	var tables = []struct {
		service     Service
		IsDestroyed bool
	}{
		{Service{Status: CreatedService}, false},
		{Service{Status: DeployedService}, false},
		{Service{Status: CreatingService}, false},
		{Service{Status: DestroyingService}, true},
		{Service{Status: DestroyedService}, true},
		{Service{Status: FailedService}, false},
	}

	for _, tt := range tables {
		if tt.service.IsDestroyed() != tt.IsDestroyed {
			t.Errorf("%s was not %t", tt.service.Status, tt.IsDestroyed)
		}
	}
}
