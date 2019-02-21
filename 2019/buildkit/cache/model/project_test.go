package model

import (
	"encoding/base64"
	"reflect"
	"testing"
)

var subnet = "subnet-123456"

func TestParseProjectSettings(t *testing.T) {
	tests := []struct {
		name        string
		settings    []byte
		want        ProjectSettings
		expectError bool
	}{
		{
			name: "valid settings demo",
			settings: []byte(`
provider:
  type: demo
administrators:
  - user1@example.com
  - user2@example.com
`),
			want: ProjectSettings{
				Administrators: []string{"user1@example.com", "user2@example.com"},
				Provider:       &Provider{Type: "demo"}},
			expectError: false,
		},
		{
			name:        "empty",
			settings:    []byte(``),
			want:        ProjectSettings{},
			expectError: true,
		},
		{
			name:        "missing list",
			settings:    []byte(`users:`),
			want:        ProjectSettings{},
			expectError: true,
		},
		{
			name: "missing users",
			settings: []byte(`
provider:
  type: k8
`),
			want:        ProjectSettings{},
			expectError: true,
		},
		{
			name: "malformed user",
			settings: []byte(`
provider:
  type: k8
users:
  - foo
`),
			want:        ProjectSettings{},
			expectError: true,
		},
		{
			name: "settings with github",
			settings: []byte(`
provider:
  type: demo
administrators:
  - user1@example.com
  - user2@example.com
github:
  linked_by: user1@example.com
`),
			want: ProjectSettings{
				Administrators: []string{"user1@example.com", "user2@example.com"},
				Provider:       &Provider{Type: "demo"},
				Github:         &Github{LinkedBy: "user1@example.com"}},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := base64.StdEncoding.EncodeToString(tt.settings)
			s, e := ParseProjectSettings(encoded)
			if tt.expectError {
				if e == nil {
					t.Errorf("didn't receive expected error")
				}

				return
			}

			if e != nil {
				t.Errorf("unexpected error: %+v", e)
				return
			}

			if s == nil {
				t.Errorf("didn't receive a settings instance: %+v %s", s, e.Message)
				return
			}

			if !reflect.DeepEqual(s.Users, tt.want.Users) {
				t.Errorf("got %+v, expected %+v", *s, tt.want)
			}

			if !reflect.DeepEqual(*s.Provider, *tt.want.Provider) {
				t.Errorf("got %+v, expected %+v", s.Provider, tt.want.Provider)
			}

			if tt.want.Github != nil {
				if s.Github == nil {
					t.Fatalf("github setting not parsed")
				}

				if tt.want.Github.LinkedBy != s.Github.LinkedBy {
					t.Errorf("got %+v, expected %+v", s.Provider, tt.want.Provider)

				}
			}
		})
	}
}

func TestParseProjectSettingsK8(t *testing.T) {

	settings :=
		[]byte(`
provider:
  type: k8
  username: username1
  password: password1
  endpoint: endpoint1
  ca_cert: ca_cert1
administrators:
- user1@example.com
users:
- user2@example.com
`)

	encoded := base64.StdEncoding.EncodeToString(settings)
	s, e := ParseProjectSettings(encoded)
	if e != nil {
		t.Fatalf("%+v", e)
	}

	if s.Provider.Type != K8 {
		t.Errorf("wrong provider parsed")
	}
	if s.Provider.Username != "username1" {
		t.Errorf("k8 username wasn't parsed")
	}
	if s.Provider.Password != "password1" {
		t.Errorf("k8 password wasn't parsed")
	}
	if s.Provider.Endpoint != "endpoint1" {
		t.Errorf("k8 username wasn't parsed")
	}
	if s.Provider.CaCert != "ca_cert1" {
		t.Errorf("k8 ca_cert wasn't parsed")
	}
}

func TestValidateProject(t *testing.T) {
	tests := []struct {
		name     string
		project  *Project
		expected AppErrorCode
	}{
		{name: "missing name", project: &Project{Model: Model{ID: "1"}}, expected: MissingName},
		{name: "invalid name", project: &Project{Model: Model{ID: "1"}, Name: "free-tier"}, expected: InvalidName},
		{name: "invalid name", project: &Project{Model: Model{ID: "1"}, Name: "name@"}, expected: InvalidName},
		{name: "invalid name", project: &Project{Model: Model{ID: "1"}, Name: "name.another"}, expected: InvalidName},
		{name: "invalid name", project: &Project{Model: Model{ID: "1"}, Name: "name_another"}, expected: InvalidName},
		{name: "valid project name with dash", project: &Project{Model: Model{ID: "1"}, Name: "name-12345", DNSName: "name-12345"}, expected: ""},
		{name: "valid project name", project: &Project{Model: Model{ID: "1"}, Name: "name12345", DNSName: "name12345"}, expected: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if tt.expected == "" {
				if err != nil {
					t.Errorf("unexpected an error: %+v", err)
				}
			} else {
				if err == nil {
					t.Errorf("was expecting an error: %s", tt.expected)
					return
				}

				if err.Code != tt.expected {
					t.Errorf("was expecting an error: %s, got %s", tt.expected, err.Code)
				}
			}
		})
	}
}
