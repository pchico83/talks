package app

import (
	"testing"
	"time"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/store"
	_ "github.com/lib/pq"
)

func TestCleanExpiredServices(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	s := Server{DB: db}

	p := &model.Project{Name: "testproject", DNSName: "testproject", Settings: demoProject}
	err := db.Create(p).Error
	if err != nil {
		t.Fatalf("error when saving project: %s", err.Error())
	}
	u := &model.User{Email: "bot@okteto.com"}
	err = db.Create(u).Error
	if err != nil {
		t.Fatalf("error when saving user: %s", err.Error())
	}
	u = &model.User{Email: "actor@okteto.com"}
	err = db.Create(u).Error
	if err != nil {
		t.Fatalf("error when saving user: %s", err.Error())
	}

	svc := &model.Service{
		Name:      "service-1",
		Manifest:  httpService,
		ProjectID: p.ID,
		IsDemo:    true,
		Status:    model.CreatedService,
	}
	err = db.Create(&svc).Error
	if err != nil {
		t.Fatalf("error when saving service: %s", err.Error())
	}
	err = db.Model(&svc).UpdateColumn("created_at", time.Now().Add(-100*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-1:  %s", err.Error())
	}

	svc = &model.Service{
		Name:      "service-2",
		Manifest:  httpService,
		ProjectID: p.ID,
		IsDemo:    true,
		Status:    model.FailedService,
	}
	err = db.Create(&svc).Error
	if err != nil {
		t.Fatalf("error when saving service: %s", err.Error())
	}
	err = db.Model(&svc).UpdateColumn("created_at", time.Now().Add(-100*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-1:  %s", err.Error())
	}

	svc = &model.Service{
		Name:      "service-3",
		Manifest:  httpService,
		ProjectID: p.ID,
		IsDemo:    true,
		Status:    model.DeployedService,
	}
	err = db.Create(&svc).Error
	if err != nil {
		t.Fatalf("error when saving service: %s", err.Error())
	}
	err = db.Model(&svc).UpdateColumn("created_at", time.Now().Add(-100*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-1:  %s", err.Error())
	}

	svc = &model.Service{
		Name:      "service-4",
		Manifest:  httpService,
		ProjectID: p.ID,
		IsDemo:    true,
		Status:    model.DestroyedService,
	}
	err = db.Create(&svc).Error
	if err != nil {
		t.Fatalf("error when saving service: %s", err.Error())
	}
	err = db.Model(&svc).UpdateColumn("created_at", time.Now().Add(-100*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-4:  %s", err.Error())
	}

	svc = &model.Service{
		Name:      "service-5",
		Manifest:  httpService,
		ProjectID: p.ID,
		IsDemo:    true,
		Status:    model.CreatedService,
	}
	err = db.Create(&svc).Error
	if err != nil {
		t.Fatalf("error when saving service: %s", err.Error())
	}

	svc = &model.Service{
		Name:      "service-6",
		Manifest:  httpService,
		ProjectID: p.ID,
		IsDemo:    false,
		Status:    model.DeployedService,
	}
	err = db.Create(&svc).Error
	if err != nil {
		t.Fatalf("error when saving service: %s", err.Error())
	}
	err = db.Model(&svc).UpdateColumn("created_at", time.Now().Add(-100*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-6:  %s", err.Error())
	}

	services := s.expiredServices()
	if len(services) != 3 {
		t.Fatalf("Wrong expired services: %v", services)
	}

	s.clean()
	time.Sleep(3 * time.Second)

	var count int
	var s1 model.Service
	if s.DB.Where("name = ?", "service-1").First(&s1).Error != nil {
		t.Fatalf("error getting service-1")
	}
	if s1.Status != model.DestroyedService {
		t.Fatalf("service-1 not destroyed: %s", s1.Status)
	}
	if s.DB.Model(&model.Activity{}).Where("service_id = ?", s1.ID).Count(&count).Error != nil {
		t.Fatalf("error getting service-1 activities")
	}
	if count != 1 {
		t.Fatalf("service-1 destroy activity not created")
	}

	var s2 model.Service
	if s.DB.Where("name = ?", "service-2").First(&s2).Error != nil {
		t.Fatalf("error getting service-2")
	}
	if s2.Status != model.DestroyedService {
		t.Fatalf("service-2 not destroyed")
	}
	if s.DB.Model(&model.Activity{}).Where("service_id = ?", s2.ID).Count(&count).Error != nil {
		t.Fatalf("error getting service-2 activities")
	}
	if count != 1 {
		t.Fatalf("service-2 destroy activity not created")
	}

	var s3 model.Service
	if s.DB.Where("name = ?", "service-3").First(&s3).Error != nil {
		t.Fatalf("error getting service-3")
	}
	if s3.Status != model.DestroyedService {
		t.Fatalf("service-3 not destroyed")
	}
	if s.DB.Model(&model.Activity{}).Where("service_id = ?", s3.ID).Count(&count).Error != nil {
		t.Fatalf("error getting service-3 activities")
	}
	if count != 1 {
		t.Fatalf("service-3 destroy activity not created")
	}

	var s4 model.Service
	if s.DB.Where("name = ?", "service-4").First(&s4).Error != nil {
		t.Fatalf("error getting service-4")
	}
	if s4.Status != model.DestroyedService {
		t.Fatalf("service-4 not destroyed")
	}
	if s.DB.Model(&model.Activity{}).Where("service_id = ?", s4.ID).Count(&count).Error != nil {
		t.Fatalf("error getting service-4 activities")
	}
	if count != 0 {
		t.Fatalf("service-4 destroy activity was created")
	}

	var s5 model.Service
	if s.DB.Where("name = ?", "service-5").First(&s5).Error != nil {
		t.Fatalf("error getting service-5")
	}
	if s5.Status != model.CreatedService {
		t.Fatalf("service-5 is destroyed")
	}
	if s.DB.Model(&model.Activity{}).Where("service_id = ?", s5.ID).Count(&count).Error != nil {
		t.Fatalf("error getting service-5 activities")
	}
	if count != 0 {
		t.Fatalf("service-5 destroy activity was created")
	}

	var s6 model.Service
	if s.DB.Where("name = ?", "service-6").First(&s6).Error != nil {
		t.Fatalf("error getting service-6")
	}
	if s6.Status != model.DeployedService {
		t.Fatalf("service-6 is destroyed")
	}
	if s.DB.Model(&model.Activity{}).Where("service_id = ?", s6.ID).Count(&count).Error != nil {
		t.Fatalf("error getting service-6 activities")
	}
	if count != 0 {
		t.Fatalf("service-6 destroy activity was created")
	}
}
