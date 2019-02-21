package app

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"bitbucket.org/okteto/okteto/backend/model"

	"bitbucket.org/okteto/okteto/backend/store"
)

const (
	httpsService = "bmFtZTogb2t0ZXRvdGVzdA0KcmVwbGljYXM6IDENCmNvbnRhaW5lcnM6DQogIG9rdGV0bzoNCiAgICBpbWFnZTogcmliZXJhcHJvamVjdC9hcHA6bGF0ZXN0DQogICAgcG9ydHM6DQogICAgICAtIGh0dHBzOjQ0MzpodHRwOjgwMDA6YXJuOmF3czphY206dXMtd2VzdC0yOmFjY291bnQ6Y2VydGlmaWNhdGUvdXVpZA0KICAgIGVudmlyb25tZW50Og0KICAgICAgLSBPS1RFVE9fQVBJX1VSTD1odHRwczovL2V4YW1wbGUuY29tL3YxDQogICAgICAtIE9LVEVUT19EQVRBQkFTRV9IT1NUPWFkYXRhYmFzZWhvc3QNCiAgICAgIC0gT0tURVRPX0RBVEFCQVNFX1BPUlQ9NTQzMg0KICAgICAgLSBPS1RFVE9fREFUQUJBU0VfVVNFUj1yaWJlcmENCiAgICAgIC0gT0tURVRPX0RBVEFCQVNFX1BBU1NXT1JEPWFwYXNzd3JvZA0KICAgICAgLSBPS1RFVE9fREFUQUJBU0VfREFUQUJBU0U9cmliZXJhDQogICAgICAtIE9LVEVUT19BUElfV0hJVEVMSVNUPVRydWUNCiAgICAgIC0gT0tURVRPX1NMQUNLX1ZFUklGSUNBVElPTl9UT0tFTj1hdG9rZW4="
	httpService  = "bmFtZTogcmliZXJhdGVzdA0KcmVwbGljYXM6IDENCmNvbnRhaW5lcnM6DQogIHJpYmVyYToNCiAgICBpbWFnZTogcmliZXJhcHJvamVjdC9hcHA6bGF0ZXN0DQogICAgcG9ydHM6DQogICAgICAtIGh0dHA6ODA6aHR0cDo4MA0KICAgIGVudmlyb25tZW50Og0KICAgICAgLSBSSUJFUkFfQVBJX1VSTD1odHRwczovL2V4YW1wbGUuY29tL3YxDQogICAgICAtIFJJQkVSQV9EQVRBQkFTRV9IT1NUPWFkYXRhYmFzZWhvc3QNCiAgICAgIC0gUklCRVJBX0RBVEFCQVNFX1BPUlQ9NTQzMg0KICAgICAgLSBSSUJFUkFfREFUQUJBU0VfVVNFUj1yaWJlcmENCiAgICAgIC0gUklCRVJBX0RBVEFCQVNFX1BBU1NXT1JEPWFwYXNzd3JvZA0KICAgICAgLSBSSUJFUkFfREFUQUJBU0VfREFUQUJBU0U9cmliZXJhDQogICAgICAtIFJJQkVSQV9BUElfV0hJVEVMSVNUPVRydWUNCiAgICAgIC0gUklCRVJBX1NMQUNLX1ZFUklGSUNBVElPTl9UT0tFTj1hdG9rZW4="
)

func TestDestroy(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		settings *model.ProjectSettings
		want     string
		wantErr  bool
	}{
		{
			name:     "service-with-https",
			manifest: httpsService,
			settings: &model.ProjectSettings{Provider: &model.Provider{Type: model.Demo}},
			wantErr:  false,
			want:     "Destroying the service 'oktetotest'...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := store.NewMemoryStore()
			defer db.Close()
			s := Server{DB: db}

			d := &model.Service{Manifest: tt.manifest}
			p := &model.Project{Model: model.Model{ID: "1-2-3-4"}, Name: "testproject", DNSName: "testproject", LoadedSettings: tt.settings}
			err := s.destroy(d, p, "activity-1")
			if (err != nil) != tt.wantErr {
				t.Errorf("destroy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			logs, err := s.getActivityLogs("activity-1")
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(logs) == 0 {
				t.Fatalf("logs were not captured")
			}

			if logs[0].Log != tt.want {
				t.Errorf("destroy() out = '%s', want '%s'", logs[0].Log, tt.want)
			}
		})
	}
}

func Test_deploy(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		settings *model.ProjectSettings
		want     string
		wantErr  bool
	}{
		{
			name:     "service-with-https",
			manifest: httpsService,
			settings: &model.ProjectSettings{Provider: &model.Provider{Type: model.Demo}},
			wantErr:  false,
			want:     "Deploying the service 'oktetotest'...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := store.NewMemoryStore()
			defer db.Close()
			s := Server{DB: db}

			d := &model.Service{Manifest: tt.manifest}
			p := &model.Project{Model: model.Model{ID: "1-2-3-4"}, Name: "testproject", DNSName: "testproject", LoadedSettings: tt.settings}
			err := s.deploy(d, p, "activity-1")
			if (err != nil) != tt.wantErr {
				t.Errorf("deploy() error = '%v', wantErr '%t'", err, tt.wantErr)
				return
			}

			logs, err := s.getActivityLogs("activity-1")
			if err != nil {
				t.Fatal(err.Error())
			}

			if len(logs) == 0 {
				t.Fatalf("logs were not captured")
			}

			if logs[0].Log != tt.want {
				t.Errorf("deploy() out = '%+v', want '%s'", logs, tt.want)
			}
		})
	}
}

func Test_buildService(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name     string
		manifest string
		wantErr  bool
	}{
		{name: "service-with-https", manifest: httpsService, wantErr: false},
		{
			name:     "service-with-http",
			manifest: httpsService,
			wantErr:  false},
		{
			name:     "service-with-no-ports",
			manifest: "bmFtZTogcmliZXJhdGVzdA0KcmVwbGljYXM6IDENCmNvbnRhaW5lcnM6DQogIHJpYmVyYToNCiAgICBpbWFnZTogcmliZXJhcHJvamVjdC9hcHA6bGF0ZXN0DQogICAgZW52aXJvbm1lbnQ6DQogICAgICAtIFJJQkVSQV9BUElfVVJMPWh0dHBzOi8vZXhhbXBsZS5jb20vdjENCiAgICAgIC0gUklCRVJBX0RBVEFCQVNFX0hPU1Q9YWRhdGFiYXNlaG9zdA0KICAgICAgLSBSSUJFUkFfREFUQUJBU0VfUE9SVD01NDMyDQogICAgICAtIFJJQkVSQV9EQVRBQkFTRV9VU0VSPXJpYmVyYQ0KICAgICAgLSBSSUJFUkFfREFUQUJBU0VfUEFTU1dPUkQ9YXBhc3N3cm9kDQogICAgICAtIFJJQkVSQV9EQVRBQkFTRV9EQVRBQkFTRT1yaWJlcmENCiAgICAgIC0gUklCRVJBX0FQSV9XSElURUxJU1Q9VHJ1ZQ0KICAgICAgLSBSSUJFUkFfU0xBQ0tfVkVSSUZJQ0FUSU9OX1RPS0VOPWF0b2tlbg==",
			wantErr:  false,
		},
		{
			name:     "service-with-no-envvar",
			manifest: "bmFtZTogcmliZXJhdGVzdA0KcmVwbGljYXM6IDENCmNvbnRhaW5lcnM6DQogIHJpYmVyYToNCiAgICBpbWFnZTogcmliZXJhcHJvamVjdC9hcHA6bGF0ZXN0",
			wantErr:  false,
		},
		{
			name:     "service-with-no-containers",
			manifest: "bmFtZTogcmliZXJhdGVzdA0KcmVwbGljYXM6IDENCmNvbnRhaW5lcnM6",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &model.Service{Manifest: tt.manifest}
			_, err := buildService(d.ID, d.Manifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_orDefault(t *testing.T) {
	tests := []struct {
		name string
		v    string
		d    string
		want string
	}{
		{name: "value", v: "value", d: "default", want: "value"},
		{name: "empty-value", v: "", d: "default", want: "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := orDefault(tt.v, tt.d); got != tt.want {
				t.Errorf("orDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveLogs(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}

	l, r := getLogger()
	d := make(chan bool, 1)
	w := &sync.WaitGroup{}
	w.Add(1)

	go s.saveLogs("activity-1", r, d, w)
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Millisecond)
			l.Printf("log-%d", i)
		}

		d <- true
	}()

	w.Wait()

	logs, err := s.getActivityLogs("activity-1")
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(logs) != 10 {
		t.Fatalf("didn't retrieve 10 log entries, got %d", len(logs))
	}

	for i := 0; i < 10; i++ {
		if logs[i].Log != fmt.Sprintf("log-%d", i) {
			t.Errorf("element %d of the array was: '%s', expected '%s'", i, logs[i].Log, fmt.Sprintf("log-%d", i))
		}
	}
}

func Test_injectSecrets(t *testing.T) {
	tests := []struct {
		name string
		s    *model.Service
		se   []*model.EnvVar
		want []*model.EnvVar
	}{
		{
			name: "no-match",
			s: &model.Service{
				Containers: map[string]*model.Container{
					"web": &model.Container{
						Environment: []*model.EnvVar{
							&model.EnvVar{Name: "ServiceEnv", Value: "ServiceValue"},
						},
					},
				},
			},
			se: []*model.EnvVar{&model.EnvVar{Name: "Secret", Value: "SecretValue"}},
			want: []*model.EnvVar{
				&model.EnvVar{Name: "ServiceEnv", Value: "ServiceValue"},
			},
		},
		{
			name: "override",
			s: &model.Service{
				Containers: map[string]*model.Container{
					"web": &model.Container{
						Environment: []*model.EnvVar{
							&model.EnvVar{Name: "Secret", Value: ""},
						},
					},
				},
			},
			se: []*model.EnvVar{&model.EnvVar{Name: "Secret", Value: "SecretValue"}},
			want: []*model.EnvVar{
				&model.EnvVar{Name: "Secret", Value: "SecretValue"},
			},
		},
		{
			name: "override-in-value",
			s: &model.Service{
				Containers: map[string]*model.Container{
					"web": &model.Container{
						Environment: []*model.EnvVar{
							&model.EnvVar{Name: "Service", Value: "$Secret"},
							&model.EnvVar{Name: "Service2", Value: "Value"},
						},
					},
				},
			},
			se: []*model.EnvVar{&model.EnvVar{Name: "Secret", Value: "SecretValue"}},
			want: []*model.EnvVar{
				&model.EnvVar{Name: "Service", Value: "SecretValue"},
				&model.EnvVar{Name: "Service2", Value: "Value"},
			},
		},
		{
			name: "some-override",
			s: &model.Service{
				Containers: map[string]*model.Container{
					"web": &model.Container{
						Environment: []*model.EnvVar{
							&model.EnvVar{Name: "Secret", Value: ""},
							&model.EnvVar{Name: "Service", Value: "ServiceValue"},
						},
					},
				},
			},
			se: []*model.EnvVar{&model.EnvVar{Name: "Secret", Value: "SecretValue"}},
			want: []*model.EnvVar{
				&model.EnvVar{Name: "Secret", Value: "SecretValue"},
				&model.EnvVar{Name: "Service", Value: "ServiceValue"},
			},
		},
		{
			name: "no-empty-envvars",
			s: &model.Service{
				Containers: map[string]*model.Container{
					"web": &model.Container{
						Environment: []*model.EnvVar{
							&model.EnvVar{Name: "Secret", Value: "ServiceOverride"},
							&model.EnvVar{Name: "Service", Value: "ServiceValue"},
						},
					},
				},
			},
			se: []*model.EnvVar{&model.EnvVar{Name: "Secret", Value: "SecretValue"}},
			want: []*model.EnvVar{
				&model.EnvVar{Name: "Secret", Value: "ServiceOverride"},
				&model.EnvVar{Name: "Service", Value: "ServiceValue"},
			},
		},
		{
			name: "no-env-in-container",
			s: &model.Service{
				Containers: map[string]*model.Container{
					"web": &model.Container{
						Environment: []*model.EnvVar{},
					},
				},
			},
			se:   []*model.EnvVar{&model.EnvVar{Name: "Secret", Value: "SecretValue"}},
			want: []*model.EnvVar{},
		},
		{
			name: "no-secrets",
			s: &model.Service{
				Containers: map[string]*model.Container{
					"web": &model.Container{
						Environment: []*model.EnvVar{
							&model.EnvVar{Name: "Service", Value: "ServiceValue"},
						},
					},
				},
			},
			se: []*model.EnvVar{},
			want: []*model.EnvVar{
				&model.EnvVar{Name: "Service", Value: "ServiceValue"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injectSecrets(tt.s, tt.se)
			if !reflect.DeepEqual(tt.want, tt.s.Containers["web"].Environment) {
				t.Logf("want: %d %+v", len(tt.want), tt.want)
				t.Logf("received: %d %+v", len(tt.s.Containers["web"].Environment), tt.s.Containers["web"].Environment)
				t.Fatalf("secrets and env was not properly merged")
			}
		})
	}
}
