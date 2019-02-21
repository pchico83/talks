package app

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/viper"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/store"
)

var demoProject = "cHJvdmlkZXI6DQogIHR5cGU6IGRlbW8NCmFkbWluaXN0cmF0b3JzOg0KICAtIHVzZXIxQGV4YW1wbGUuY29t"

func TestCreatService(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	svc := &model.Service{Manifest: httpsService}
	p := &model.Project{Model: model.Model{ID: "1-2-3-4"}, Name: "testproject", DNSName: "testproject", Settings: demoProject}
	u := &model.User{Email: "actor@example.com"}
	db.Create(u)

	created, err := s.CreateService(p, svc, u)
	if err != nil {
		t.Fatalf("Create failed %+v", err)
	}

	if created.ID == "" {
		t.Error("ID wasn't set")
	}

	if created.UpdatedAt.IsZero() {
		t.Error("UpdatedAt wasn't set")
	}

	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt wasn't set")
	}

	get, err := s.GetServiceAndActivities(p, created.ID)
	if err != nil {
		t.Fatalf("Get failed service-%s: %+v", created.ID, err)
	}

	if get.ID != created.ID {
		t.Fatalf("received a different service")
	}

	if get.Links.Activities == "" || get.Links.Self == "" {
		t.Fatalf("Links where not set")
	}

	if get.Activities == nil {
		t.Fatalf("Activities where not set")
	}

	if get.Activities[0].ActorEmail != "actor@example.com" {
		t.Errorf("actor email was not set")
	}
}

func TestCreatServiceQuota(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	p := &model.Project{Model: model.Model{ID: "1-2-3-4"}, Name: "testproject", DNSName: "testproject", Settings: demoProject}
	u := &model.User{Email: "actor@example.com"}
	db.Create(u)

	svc := &model.Service{Manifest: httpsService, Name: "service-1", CreatedBy: u.ID, IsDemo: true}
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-2"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-3"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-4"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-5"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-6"
	if _, err := s.CreateService(p, svc, u); err == nil {
		t.Fatalf("Create didn't fail due to quota %+v", err)
	}

}

func TestCreatServiceQuotaNoForOktetos(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	p := &model.Project{Model: model.Model{ID: "1-2-3-4"}, Name: "testproject", DNSName: "testproject", Settings: demoProject}
	u := &model.User{Email: "actor@okteto.com"}
	db.Create(u)

	svc := &model.Service{Manifest: httpsService, Name: "service-1", CreatedBy: u.ID, IsDemo: true}
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-2"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-3"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-4"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-5"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}
	svc.Name = "service-6"
	if _, err := s.CreateService(p, svc, u); err != nil {
		t.Fatalf("Create failed %+v", err)
	}

}

func TestUpdateService(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	svc := &model.Service{Manifest: httpsService, Name: "service"}
	p := &model.Project{Model: model.Model{ID: "1-2-3-4"}, Name: "testproject", DNSName: "testproject", Settings: demoProject}

	u, _ := s.GetOrCreateUser("user@example.com", true, "")
	created, err := s.CreateService(p, svc, u)
	if err != nil {
		t.Fatalf("Create failed %+v", err)
	}

	s.UpdateManifest(p.ID, created.ID, httpService, u.ID, "manifest updated")
	if err != nil {
		t.Fatalf("Update failed %+v", err)
	}

	get, err := s.GetServiceAndActivities(p, created.ID)
	if err != nil {
		t.Fatalf("Get failed service-%s: %+v", created.ID, err)
	}

	if get.Name != "riberatest" {
		t.Errorf("name wasn't updated. Was expecting 'riberatest' but got '%s'", get.Name)
	}
	if get.Manifest != httpService {
		t.Fatalf("Manifest wasn't updated. Expected\r\n%s\r\nGot\r\n%s", httpService, get.Manifest)
	}

	if len(get.Activities) != 2 {
		t.Fatalf("missing activities")
	}

	if get.Activities[1].Type != model.Updated {
		t.Fatalf("activty wasn't updated, it was: %s", get.Activities[1].Type)
	}

	if get.Activities[1].ActorEmail != u.Email {
		t.Errorf("update activity didn't have the actor email")
	}

	logs, appErr := s.GetActivityLogs(p, created.ID, get.Activities[1].ID)
	if appErr != nil {
		t.Fatal(appErr.Error())
	}

	if logs[0].Log != "manifest updated" {
		t.Errorf("update activity didn't have a log")
	}

}

func TestGetServices(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	p := &model.Project{Model: model.Model{ID: "project-1"}, Name: "testproject", DNSName: "testproject", Settings: demoProject}
	p.LoadedSettings, _ = model.ParseProjectSettings(p.Settings)
	p2 := &model.Project{Model: model.Model{ID: "project-2"}, Name: "testproject-2", DNSName: "testproject-2", Settings: demoProject}
	p2.LoadedSettings, _ = model.ParseProjectSettings(p2.Settings)

	svc := &model.Service{Manifest: httpsService, Name: "service-1"}

	u := &model.User{}

	_, appErr := s.CreateService(p, svc, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	svc2 := &model.Service{Manifest: httpService, Name: "service-2"}
	_, appErr = s.CreateService(p, svc2, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	svc3 := &model.Service{Manifest: httpService, Name: "service-3"}
	_, appErr = s.CreateService(p2, svc3, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	svc4 := &model.Service{Manifest: httpService, Name: "service-4"}
	_, appErr = s.CreateService(p, svc4, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	appErr = s.DeleteService(p, svc4.ID, u)
	if appErr != nil {
		t.Fatalf("Delete failed %+v", appErr)
	}

	err := s.waitUntil(p, svc4.ID, model.DestroyedService)
	if err != nil {
		t.Fatalf("Delete failed %s", err)
	}

	result := db.Model(&svc4).UpdateColumn("updated_at", svc4.UpdatedAt.Add(-65*time.Minute))
	if result.Error != nil {
		t.Fatalf("Update to make deleted service older failed %s", result.Error.Error())
	}

	services, appErr := s.GetServices(p.ID, false)
	if appErr != nil {
		t.Fatalf("GetServices failed: %+v", appErr)
	}

	if len(services) != 2 {
		t.Errorf("didn't received 2 services, got %d", len(services))
		for _, s := range services {
			t.Errorf("%s, status: %s, updated_at: %s", s.Name, s.Status, s.UpdatedAt)
		}
	}

	for _, s := range services {
		if s.ProjectID != p.ID {
			t.Errorf("user returned was from a different project")
		}
	}

	services, appErr = s.GetServices(p.ID, true)
	if appErr != nil {
		t.Fatalf("Get services with deleted %+v", appErr)
	}

	if len(services) != 3 {
		t.Fatalf("didn't received 3 services, got %d: %+v", len(services), services)
	}
}

func TestGetServiceAndActivities(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	svc := &model.Service{Manifest: httpsService, Name: "service"}
	p := &model.Project{Model: model.Model{ID: "project-1"}, Name: "testproject", DNSName: "testproject", Settings: demoProject}
	p.LoadedSettings, _ = model.ParseProjectSettings(p.Settings)

	u := &model.User{Email: "actor@example.com"}

	s.DB.Create(&u)

	created, appErr := s.CreateService(p, svc, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	appErr = s.StartService(p, created.ID, u)
	if appErr != nil {
		t.Fatalf("Start failed %+v", appErr)
	}

	err := s.waitUntil(p, svc.ID, model.DeployedService)
	if err != nil {
		t.Fatalf("Start failed %s", err)
	}

	get, appErr := s.GetServiceAndActivities(p, created.ID)
	if appErr != nil {
		t.Fatalf("Start failed %+v", appErr)
	}

	if len(get.Activities) != 2 {
		t.Fatalf("Missing activities %+v", get.Activities)
	}

	if get.Activities[0].Type != model.Created {
		t.Fatalf("First activity wasn't created: %+v", get.Activities[0])
	}

	if get.Activities[0].ActorEmail != "actor@example.com" {
		t.Errorf("First activity is missing the actor: %+v", get.Activities[0])
	}

	if get.Activities[1].Type != model.Deployed {
		t.Fatalf("First activity wasn't created: %+v", get.Activities[0])
	}

	if get.Activities[1].ActorEmail != "actor@example.com" {
		t.Errorf("Second activity is missing the actor: %+v", get.Activities[0])
	}
}

func TestServiceLifecycle(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	svc := &model.Service{Manifest: httpsService, Name: "service"}
	p := &model.Project{Model: model.Model{ID: "project-1"}, Name: "testproject", DNSName: "testproject", Settings: demoProject}
	u := &model.User{Email: "user@example.com"}
	db.Create(&u)

	_, appErr := s.CreateService(p, svc, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	created, appErr := s.GetServiceAndActivities(p, svc.ID)
	if appErr != nil {
		t.Fatalf("Get failed %+v", appErr)
	}

	if len(created.Activities) != 1 {
		t.Fatalf("New service didn't contain 1 activities, got %d", len(created.Activities))
	}

	appErr = s.StartService(p, svc.ID, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	err := s.waitUntil(p, svc.ID, model.DeployedService)
	if err != nil {
		t.Fatalf("get failed %s", err.Error())
	}

	appErr = s.DeleteService(p, svc.ID, u)
	if appErr != nil {
		t.Fatalf("Delete failed %+v", appErr)
	}

	err = s.waitUntil(p, svc.ID, model.DestroyedService)
	if err != nil {
		t.Fatalf("Delete failed %s", err.Error())
	}

	finalized, appErr := s.GetServiceAndActivities(p, svc.ID)
	if appErr != nil {
		t.Fatalf("Get failed %+v", appErr)
	}

	if len(finalized.Activities) != 3 {
		t.Fatalf("Final service didn't contain 3 activities, got %d", len(finalized.Activities))
	}

	logs, err := s.getActivityLogs(finalized.Activities[1].ID)
	if err != nil {
		t.Fatalf("failed to get logs: %s", err)
	}

	if len(logs) != 3 {
		t.Errorf("Deployment logs didn't contain 10 logs, got %+v", logs)
	}

	logs, err = s.getActivityLogs(finalized.Activities[2].ID)
	if err != nil {
		t.Fatalf("failed to get logs: %s", err)
	}

	if len(logs) != 2 {
		t.Errorf("Destroy logs didn't contain 10 logs, got %+v", logs)
	}
}

func TestDevMode(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	svc := &model.Service{Manifest: httpsService, Name: "service"}
	p := &model.Project{Model: model.Model{ID: "project-1"}, Name: "testproject", DNSName: "testproject", Settings: demoProject}
	u := &model.User{Email: "user@example.com"}
	db.Create(&u)

	_, appErr := s.CreateService(p, svc, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	created, appErr := s.GetServiceAndActivities(p, svc.ID)
	if appErr != nil {
		t.Fatalf("Get failed %+v", appErr)
	}

	if len(created.Activities) != 1 {
		t.Fatalf("New service didn't contain 1 activities, got %d", len(created.Activities))
	}

	appErr = s.StartService(p, svc.ID, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	err := s.waitUntil(p, svc.ID, model.DeployedService)
	if err != nil {
		t.Fatalf("get failed %s", err.Error())
	}

	appErr = s.EnableDevMode(p, svc.ID, u)
	if appErr != nil {
		t.Fatalf("Enable dev mode failed %+v", appErr)
	}

	err = s.waitUntil(p, svc.ID, model.DevDeployedService)
	if err != nil {
		t.Fatalf("enabling dev deploy failed %s", err.Error())
	}

	dev, appErr := s.getService(p, svc.ID)
	if appErr != nil {
		t.Fatalf("get failed %+v", appErr)
	}

	if !dev.Dev {
		t.Fatalf("service wasn't marked as dev %+v", dev)
	}

	appErr = s.StartService(p, svc.ID, u)
	if appErr != nil {
		t.Fatalf("Create failed %+v", appErr)
	}

	err = s.waitUntil(p, svc.ID, model.DeployedService)
	if err != nil {
		t.Fatalf("deploy failed %s", err.Error())
	}

	dev, appErr = s.getService(p, svc.ID)
	if appErr != nil {
		t.Fatalf("get failed %+v", appErr)
	}

	if dev.Dev {
		t.Fatalf("service was still marked as dev after deployment %+v", dev)
	}
}

func (s *Server) waitUntil(project *model.Project, serviceID string, expected model.ServiceStatus) error {
	var lastStatus model.ServiceStatus
	for i := 0; i < 100; i++ {
		get, appErr := s.getService(project, serviceID)
		if appErr != nil {
			return appErr

		}

		if get.Status == expected {
			return nil
		}

		lastStatus = get.Status
		time.Sleep(10 * time.Millisecond)
	}

	return fmt.Errorf("service never became %s, it was: %s", expected, lastStatus)
}

func TestAddLog(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}

	for i := 0; i < 100; i++ {
		err := s.addLog("activity-1", fmt.Sprintf("log-%d", i))
		if err != nil {
			t.Fatalf(err.Error())
		}
	}

	logs, err := s.getActivityLogs("activity-1")
	if err != nil {
		t.Fatalf(err.Error())
	}

	if len(logs) != 100 {
		t.Fatalf("missing logs. Got %d was expecting 100", len(logs))
	}

	for i := 0; i < 100; i++ {
		if logs[i].Log != fmt.Sprintf("log-%d", i) {
			t.Errorf("disordered logs. Got %s was expecting log-%d", logs[1].Log, i)
		}
	}
}

func TestBuildServiceEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		service  model.Service
		project  model.Project
		expected []string
	}{
		{
			name: "internal-ports",
			service: model.Service{
				Name: "test",
				Containers: map[string]*model.Container{
					"test": &model.Container{
						Expose: []string{"80"},
					},
				},
			},
			project: model.Project{
				LoadedSettings: &model.ProjectSettings{
					Provider: &model.Provider{
						Ingress: &model.IngressController{},
					},
				},
			},
			expected: []string{},
		},
		{
			name: "load-balancer-ports",
			service: model.Service{
				Name: "test",
				Containers: map[string]*model.Container{
					"test": &model.Container{
						Ports: []string{"443", "80", "5000"},
					},
				},
			},
			project: model.Project{
				LoadedSettings: &model.ProjectSettings{
					Provider: &model.Provider{
						Ingress: &model.IngressController{},
					},
				},
			},
			expected: []string{
				"https://test.project.okteto.net",
				"http://test.project.okteto.net",
				"test.project.okteto.net:5000",
			},
		},
		{
			name: "ingress-ports",
			service: model.Service{
				Name: "test",
				Containers: map[string]*model.Container{
					"test": &model.Container{
						Ingress: []*model.Ingress{
							&model.Ingress{
								Host: "host",
								Port: "33",
							},
							&model.Ingress{
								Host: model.ProjectName,
								Port: "22",
							},
						},
					},
				},
			},
			project: model.Project{
				Name:    "pp",
				DNSName: "pp-1",
				LoadedSettings: &model.ProjectSettings{
					Provider: &model.Provider{
						Ingress: &model.IngressController{
							Domain: "ingress.okteto.net",
						},
					},
				},
			},
			expected: []string{
				"http://host.ingress.okteto.net",
				"http://pp.ingress.okteto.net",
			},
		},
		{
			name: "ingress-tls-ports",
			service: model.Service{
				Name: "test",
				Containers: map[string]*model.Container{
					"test": &model.Container{
						Ingress: []*model.Ingress{
							&model.Ingress{
								Host: "host",
								Port: "33",
							},
							&model.Ingress{
								Host: model.ProjectName,
								Port: "22",
							},
						},
					},
				},
			},
			project: model.Project{
				Name:    "pp",
				DNSName: "pp-1",
				LoadedSettings: &model.ProjectSettings{
					Provider: &model.Provider{
						Ingress: &model.IngressController{
							AppendProject: true,
							Domain:        "ingress.okteto.net",
							TLS:           &model.TLS{},
						},
					},
				},
			},
			expected: []string{
				"https://host-dot-pp-1.ingress.okteto.net",
				"https://pp-dot-pp-1.ingress.okteto.net",
			},
		},
	}

	server := Server{}
	dns := "project.okteto.net"
	viper.Set("aws.access_key", "test")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.buildServiceEndpoints(&tt.service, &tt.project, dns)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("buildServiceEndpoints(): %+v, expected: %+v", result, tt.expected)
			}

		})
	}

}
