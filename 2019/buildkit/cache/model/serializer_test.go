package model

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestUnmarshalService(t *testing.T) {
	var tests = []struct {
		raw      []byte
		expected *Service
	}{
		{
			raw: []byte(`
name: test
replicas: 2
volumes:
  data1:
  data2:
    persistent: true
containers:
  nginx:
    image: nginx:alpine
    command: run
    ports:
    - 80
    ingress:
    - host: in
      path: /api
      port: 8080
    expose:
    - 5000
    environment:
    - foo=bar`),
			expected: &Service{
				Name:        "test",
				Replicas:    2,
				GracePeriod: 30,
				Volumes: map[string]*Volume{
					"data1": &Volume{
						Name:       "data1",
						Persistent: false,
						Size:       "",
					},
					"data2": &Volume{
						Name:       "data2",
						Persistent: true,
						Size:       "20Gi",
					},
				},
				Containers: map[string]*Container{
					"nginx": &Container{
						Image:   "nginx:alpine",
						Command: "run",
						Environment: []*EnvVar{
							{
								Name:  "foo",
								Value: "bar",
							},
						},
						Ports: []string{"80"},
						Ingress: []*Ingress{
							&Ingress{
								Host: "in",
								Path: "/api",
								Port: "8080",
							},
						},
						Expose: []string{"5000"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		var s Service
		err := yaml.Unmarshal(tt.raw, &s)
		if err != nil {
			t.Fatalf(err.Error())
		}

		if !reflect.DeepEqual(&s, tt.expected) {
			t.Errorf("Expected: %+v \n Received: %+v", tt.expected, &s)
		}
	}
}

func TestTranslatePorts(t *testing.T) {
	var tests = []struct {
		service  Service
		expected Service
	}{
		{
			service: Service{
				Name: "test",
				Containers: map[string]*Container{
					"nginx": &Container{
						Ports: []string{"https:443:http:80"},
						Ingress: []*Ingress{
							{
								Port: "8080",
							},
						},
					},
				},
			},
			expected: Service{
				Name: "test",
				Containers: map[string]*Container{
					"nginx": &Container{
						Ports: []string{"80"},
						Ingress: []*Ingress{
							{
								Host: "test",
								Path: "/",
								Port: "8080",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt.service.translatePorts()
		if !reflect.DeepEqual(tt.service, tt.expected) {
			t.Errorf("Expected: %+v \n Received: %+v", tt.expected, tt.service)
		}
	}
}

func TestMarshalEnvVar(t *testing.T) {
	tests := []struct {
		name    string
		env     *EnvVar
		want    string
		wantErr bool
	}{
		{
			name:    "happy-path",
			env:     &EnvVar{Name: "foo", Value: "bar"},
			want:    "foo=bar\n",
			wantErr: false,
		},
		{
			name:    "empty-var",
			env:     &EnvVar{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing-value",
			env:     &EnvVar{Name: "foo"},
			want:    "foo=\n",
			wantErr: false,
		},
		{
			name:    "missing-name",
			env:     &EnvVar{},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := yaml.Marshal(tt.env)

			if !tt.wantErr && err != nil {
				t.Errorf("envvar.MarshalYaml failed: %s", err.Error())
			}

			if string(got[:]) != tt.want {
				t.Errorf("got '%s' \n want '%s'", string(got[:]), tt.want)
			}

		})
	}
}

func TestUnmarshalEnvVar(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		want    *EnvVar
		wantErr bool
	}{
		{
			name:    "happy-path",
			env:     "foo=bar",
			want:    &EnvVar{Name: "foo", Value: "bar"},
			wantErr: false,
		},
		{
			name:    "empty-string",
			env:     "",
			want:    &EnvVar{},
			wantErr: true,
		},
		{
			name:    "missing-equal",
			env:     "foo",
			want:    &EnvVar{},
			wantErr: true,
		},
		{
			name:    "missing-value",
			env:     "foo=",
			want:    &EnvVar{Name: "foo"},
			wantErr: false,
		},
		{
			name:    "missing-name",
			env:     "=bar",
			want:    &EnvVar{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var got EnvVar
			err := yaml.Unmarshal([]byte(tt.env), &got)

			if !tt.wantErr && err != nil {
				t.Errorf("envvar.UnmarshalYaml failed: %s", err.Error())
			}

			if !reflect.DeepEqual(&got, tt.want) {
				t.Errorf("got %+v \n want %+v", got, tt.want)
			}
		})
	}
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest []byte
		expected AppErrorCode
	}{
		{
			name:     "empty",
			manifest: []byte(``),
			expected: MissingName,
		},
		{
			name:     "only-name",
			manifest: []byte(`name: silent-sky`),
			expected: InvalidContainerCount,
		},
		{
			name: "no-containers",
			manifest: []byte(`
name: silent-sky
containers:
`),
			expected: InvalidContainerCount,
		},
		{
			name: "no-container-image",
			manifest: []byte(`
name: silent-sky
containers:
  welcome:
`),
			expected: MissingContainerImage,
		},
		{
			name: "empty-image",
			manifest: []byte(`
name: silent-sky
containers:
  welcome:
    image: 
`),
			expected: MissingContainerImage,
		},
		{
			name: "minimum",
			manifest: []byte(`
name: silent-sky
containers:
  site:
    image: okteto/welcome
`),
			expected: "",
		},
		{
			name: "empty-lists",
			manifest: []byte(`
name: silent-sky
containers:
  site:
    image: okteto/welcome
    environment:
    ports:
    expose:
    ingress:
    resources:
`),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, appErr := parseManifest(tt.manifest)
			if tt.expected == "" {
				if appErr != nil {
					t.Errorf("Unexpected error: %s,  %s", appErr.Code, appErr.Message)

				}
			} else {

				if appErr.Code != tt.expected {
					t.Errorf("Expected: %s, got %s: %s", tt.expected, appErr.Code, appErr.Message)
				}
			}
		})
	}
}
