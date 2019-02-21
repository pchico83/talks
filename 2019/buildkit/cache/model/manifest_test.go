package model

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestParseEncodedManifest(t *testing.T) {
	manifest := "bmFtZTogb2t0ZXRvdGVzdApyZXBsaWNhczogMTAKY29udGFpbmVyczoKICBva3RldG86CiAgICBpbWFnZTogb2t0ZXRvL2FwaTpsYXRlc3QKICAgIHBvcnRzOgogICAgICAtIGh0dHA6ODA6aHR0cDo4MDAwCiAgICBlbnZpcm9ubWVudDoKICAgICAgLSBPS1RFVE9fQVBJX1VSTD1odHRwczovL2V4YW1wbGUuY29tL3YxCiAgICAgIC0gT0tURVRPX0RBVEFCQVNFX0hPU1Q9YWRhdGFiYXNlaG9zdAogICAgICAtIE9LVEVUT19EQVRBQkFTRV9QT1JUPTU0MzIKICAgICAgLSBPS1RFVE9fREFUQUJBU0VfVVNFUj1yaWJlcmEKICAgICAgLSBPS1RFVE9fREFUQUJBU0VfUEFTU1dPUkQ9YXBhc3N3b3JkCiAgICAgIC0gT0tURVRPX0RBVEFCQVNFX0RBVEFCQVNFPXJpYmVyYQogICAgICAtIE9LVEVUT19BUElfV0hJVEVMSVNUPVRydWUKICAgICAgLSBPS1RFVE9fU0xBQ0tfVkVSSUZJQ0FUSU9OX1RPS0VOPWF0b2tlbg=="
	s, err := ParseEncodedManifest(manifest)
	if err != nil {
		t.Errorf("failed to parse the manifest: %s", err.Message)
	}

	if s == nil {
		t.Errorf("failed to parse the manifest, the service is nil")
	}

	if s.Name == "" {
		t.Errorf("failed to parse the name")
	}

	if s.Name != "oktetotest" {
		t.Errorf("failed to parse the name, got %s instead of %s", s.Name, "oktetotest")
	}

	if s.Replicas != 10 {
		t.Errorf("failed to parse the replicas, got %d instead of %d", s.Replicas, 10)
	}

	if len(s.Containers) != 1 {
		t.Errorf("failed to parse the containers: %+v", s)
	}

	if s.Containers["okteto"].Image != "okteto/api:latest" {
		t.Errorf("failed to parse the container's image: %+v", s)
	}

	if s.Containers["okteto"].Ports[0] != "8000" {
		t.Errorf("failed to parse the container's port: %+v", s.Containers["okteto"].Ports[0])
	}
}

func TestParseEncodedManifestMissingName(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		err      AppErrorCode
	}{
		{"missing name", "cmVwbGljYXM6IDENCmNvbnRhaW5lcnM6DQogIG9rdGV0bzoNCiAgICBpbWFnZTogb2t0ZXRvL2FwcDpsYXRlc3QNCiAgICBwb3J0czoNCiAgICAgIC0gaHR0cHM6NDQzOmh0dHA6ODAwMDphcm46YXdzOmFjbTp1cy13ZXN0LTI6YWNjb3VudDpjZXJ0aWZpY2F0ZS91dWlkDQogICAgZW52aXJvbm1lbnQ6DQogICAgICAtIE9LVEVUT19BUElfVVJMPWh0dHBzOi8vZXhhbXBsZS5jb20vdjENCiAgICAgIC0gT0tURVRPX0RBVEFCQVNFX0hPU1Q9YWRhdGFiYXNlaG9zdA0KICAgICAgLSBPS1RFVE9fREFUQUJBU0VfUE9SVD01NDMyDQogICAgICAtIE9LVEVUT19EQVRBQkFTRV9VU0VSPXJpYmVyYQ0KICAgICAgLSBPS1RFVE9fREFUQUJBU0VfUEFTU1dPUkQ9YXBhc3N3cm9kDQogICAgICAtIE9LVEVUT19EQVRBQkFTRV9EQVRBQkFTRT1yaWJlcmENCiAgICAgIC0gT0tURVRPX0FQSV9XSElURUxJU1Q9VHJ1ZQ0KICAgICAgLSBPS1RFVE9fU0xBQ0tfVkVSSUZJQ0FUSU9OX1RPS0VOPWF0b2tlbg==", MissingName},
		{"bad name", "bmFtZTogbmFtZV93aXRoJnN5bWJvbHMNCnJlcGxpY2FzOiAxDQpjb250YWluZXJzOg0KICBva3RldG86DQogICAgaW1hZ2U6IG9rdGV0by9hcHA6bGF0ZXN0DQogICAgcG9ydHM6DQogICAgICAtIGh0dHBzOjQ0MzpodHRwOjgwMDA6YXJuOmF3czphY206dXMtd2VzdC0yOmFjY291bnQ6Y2VydGlmaWNhdGUvdXVpZA0KICAgIGVudmlyb25tZW50Og0KICAgICAgLSBPS1RFVE9fQVBJX1VSTD1odHRwczovL2V4YW1wbGUuY29tL3YxDQogICAgICAtIE9LVEVUT19EQVRBQkFTRV9IT1NUPWFkYXRhYmFzZWhvc3QNCiAgICAgIC0gT0tURVRPX0RBVEFCQVNFX1BPUlQ9NTQzMg0KICAgICAgLSBPS1RFVE9fREFUQUJBU0VfVVNFUj1yaWJlcmENCiAgICAgIC0gT0tURVRPX0RBVEFCQVNFX1BBU1NXT1JEPWFwYXNzd3JvZA0KICAgICAgLSBPS1RFVE9fREFUQUJBU0VfREFUQUJBU0U9cmliZXJhDQogICAgICAtIE9LVEVUT19BUElfV0hJVEVMSVNUPVRydWUNCiAgICAgIC0gT0tURVRPX1NMQUNLX1ZFUklGSUNBVElPTl9UT0tFTj1hdG9rZW4=", InvalidName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseEncodedManifest(tt.manifest)
			if err == nil {
				t.Fatalf("parse didn't fail, got %+v", s)
			}

			if err.Code != tt.err {
				t.Errorf("parse returned the wrong error, got %+v, expected %s", err, tt.err)
			}
		})
	}
}

func Test_translateYamlError(t *testing.T) {
	var tests = []struct {
		name     string
		err      *yaml.TypeError
		expected map[string]string
	}{
		{
			name:     "sequence-for-map",
			err:      &yaml.TypeError{Errors: []string{"line 4: cannot unmarshal !!seq into model.Container"}},
			expected: map[string]string{"line": "4", "expected": "map", "received": "sequence"},
		},
		{
			name:     "map-for-sequence",
			err:      &yaml.TypeError{Errors: []string{"line 6: cannot unmarshal !!map into []*model.EnvVar"}},
			expected: map[string]string{"line": "6", "expected": "sequence", "received": "map"},
		},
		{
			name:     "sequence-for-string",
			err:      &yaml.TypeError{Errors: []string{"line 5: cannot unmarshal !!seq into string"}},
			expected: map[string]string{"line": "5", "expected": "string", "received": "sequence"},
		},
		{
			name:     "string-for-sequence",
			err:      &yaml.TypeError{Errors: []string{"line 4: cannot unmarshal !!str `hello` into []*model.EnvVar"}},
			expected: map[string]string{"line": "4", "expected": "sequence", "received": "string"},
		},
		{
			name:     "string-for-bool",
			err:      &yaml.TypeError{Errors: []string{"line 4: cannot unmarshal !!str `aha` into bool"}},
			expected: map[string]string{"line": "4", "expected": "boolean", "received": "string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := translateYamlTypeError(tt.err)
			if !reflect.DeepEqual(appErr.Data, tt.expected) {
				t.Errorf("expected: %+v got %+v", tt.expected, appErr.Data)
			}
		})
	}
}
