package model

import (
	"io/ioutil"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/stretchr/testify/require"
)

func TestValidateEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		e           *Environment
		expectError bool
	}{
		{
			name:        "basic",
			expectError: true,
			e: &Environment{
				Name: "test",
			},
		},
		{
			name:        "demo",
			expectError: false,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type: Demo,
				},
			},
		},
		{
			name:        "k8-empty",
			expectError: true,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
				},
			},
		},
		{
			name:        "k8-basic",
			expectError: false,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
					Password: "password",
					Endpoint: "localhost",
					CaCert:   "ca-cert",
				},
			},
		},
		{
			name:        "k8-ingress-no-domain",
			expectError: true,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
					Password: "password",
					Endpoint: "localhost",
					CaCert:   "ca-cert",
					Ingress:  &IngressController{},
				},
			},
		},
		{
			name:        "k8-ingress-ok-domain",
			expectError: false,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
					Password: "password",
					Endpoint: "localhost",
					CaCert:   "ca-cert",
					Ingress:  &IngressController{Domain: "example.com"},
				},
			},
		},
		{
			name:        "k8-ingress-ok-letscrypt",
			expectError: false,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
					Password: "password",
					Endpoint: "localhost",
					CaCert:   "ca-cert",
					Ingress: &IngressController{
						Domain: "example.com",
						TLS: &TLS{
							Type: LetsEncrypt,
						},
					},
				},
			},
		},
		{
			name:        "k8-ingress-ko-tls",
			expectError: true,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
					Password: "password",
					Endpoint: "localhost",
					CaCert:   "ca-cert",
					Ingress: &IngressController{
						Domain: "example.com",
						TLS:    &TLS{},
					},
				},
			},
		},
		{
			name:        "k8-ingress-ok-fix",
			expectError: false,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
					Password: "password",
					Endpoint: "localhost",
					CaCert:   "ca-cert",
					Ingress: &IngressController{
						Domain: "example.com",
						TLS: &TLS{
							Type: FixCertificate,
							Certificate: &Certificate{
								Secret:    "secret",
								Namespace: "namespace",
							},
						},
					},
				},
			},
		},
		{
			name:        "k8-ingress-ko-secret-fix",
			expectError: true,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
					Password: "password",
					Endpoint: "localhost",
					CaCert:   "ca-cert",
					Ingress: &IngressController{
						Domain: "example.com",
						TLS: &TLS{
							Type: FixCertificate,
							Certificate: &Certificate{
								Namespace: "namespace",
							},
						},
					},
				},
			},
		},
		{
			name:        "k8-ingress-ko-fix",
			expectError: true,
			e: &Environment{
				Name: "test",
				Provider: &Provider{
					Type:     K8,
					Username: "username",
					Password: "password",
					Endpoint: "localhost",
					CaCert:   "ca-cert",
					Ingress: &IngressController{
						Domain: "example.com",
						TLS: &TLS{
							Type: FixCertificate,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.e.Validate()
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMarshalEnvironment(t *testing.T) {
	readBytes, err := ioutil.ReadFile("./examples/environment.yml")
	require.NoError(t, err)
	var envStruct Environment
	err = yaml.Unmarshal(readBytes, &envStruct)
	require.NoError(t, err)
	envBytes, err := yaml.Marshal(envStruct)
	require.NoError(t, err)
	if string(envBytes) != string(readBytes) {
		t.Fatal(string(envBytes))
	}
}

func TestDockerConfig(t *testing.T) {
	e := &Environment{}
	b64DockerConfig := e.B64DockerConfig()
	if b64DockerConfig != "" {
		t.Fatal("Expected empty encoded value")
	}

	e.Registry = &Registry{
		Username: "username",
		Password: "password",
	}
	b64DockerConfig = e.B64DockerConfig()
	expectedValue := "CnsKCSJhdXRocyI6IHsKCQkiaHR0cHM6Ly9pbmRleC5kb2NrZXIuaW8vdjEvIjogewoJCQkiYXV0aCI6ICJkWE5sY201aGJXVTZjR0Z6YzNkdmNtUT0iCgkJfQoJfQp9Cg=="
	if b64DockerConfig != expectedValue {
		t.Fatalf("Wrong encoded value: %s", b64DockerConfig)
	}
}
