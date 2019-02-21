package app

import (
	"testing"
	"time"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/store"
	_ "github.com/lib/pq"
)

func TestStuckActivities(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	s := Server{DB: db}

	svc := &model.Service{
		Manifest:  "manifest",
		ProjectID: "project-id",
		Status:    model.DestroyedService,
	}

	err := db.Save(&svc).Error
	if err != nil {
		t.Fatalf("error when saving service: %s", err.Error())
	}

	//activity-1 is not in progress
	a1 := &model.Activity{
		ServiceID: svc.ID,
		Status:    model.Completed,
		Type:      model.Destroyed,
	}
	err = db.Save(&a1).Error
	if err != nil {
		t.Fatalf("error when saving activity-1: %s", err.Error())
	}
	err = db.Model(&a1).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update activity-1:  %s", err.Error())
	}

	//activity-2 is in progress less than 15 min
	a2 := &model.Activity{
		ServiceID: svc.ID,
		Status:    model.InProgress,
		Type:      model.Destroyed,
	}
	err = db.Save(&a2).Error
	if err != nil {
		t.Fatalf("error when saving activity-2: %s", err.Error())
	}
	err = db.Model(&a2).UpdateColumn("updated_at", time.Now().Add(-10*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update activity-1=2:  %s", err.Error())
	}

	//activity-3 is in progress more than 15 min but it is generating logs
	a3 := &model.Activity{
		ServiceID: svc.ID,
		Status:    model.InProgress,
		Type:      model.Destroyed,
	}
	err = db.Save(&a3).Error
	if err != nil {
		t.Fatalf("error when saving activity-3: %s", err.Error())
	}
	err = db.Model(&a3).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update activity-3:  %s", err.Error())
	}
	l1 := &model.ActivityLog{
		ActivityID: a3.ID,
		Log:        "this line",
	}
	err = db.Save(&l1).Error
	if err != nil {
		t.Fatalf("error when saving log-1: %s", err.Error())
	}

	//activity-4 is in progress more than 15 min and it is not generating logs
	a4 := &model.Activity{
		ServiceID: svc.ID,
		Status:    model.InProgress,
		Type:      model.Destroyed,
	}
	err = db.Save(&a4).Error
	if err != nil {
		t.Fatalf("error when saving activity-4: %s", err.Error())
	}
	err = db.Model(&a4).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update activity-4:  %s", err.Error())
	}
	l2 := &model.ActivityLog{
		ActivityID: a4.ID,
		Log:        "this line",
	}
	err = db.Save(&l2).Error
	if err != nil {
		t.Fatalf("error when saving log-2: %s", err.Error())
	}
	err = db.Model(&l2).UpdateColumn("created_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update log-2:  %s", err.Error())
	}

	activities := s.stuckActivities()
	if len(activities) != 1 {
		t.Fatalf("Wrong stuck activities: %v", activities)
	}
	if activities[0].ID != a4.ID {
		t.Fatalf("Wrong stuck activity: %v", activities[0])
	}

	updated := a3.UpdatedAt
	err = s.DB.Where("id = ?", a3.ID).First(&a3).Error
	if err != nil {
		t.Fatalf("failed to get activity-3:  %s", err.Error())
	}
	if a3.UpdatedAt == updated {
		t.Fatalf("activity-3 updated field not refreshed: %s vs %s", a3.UpdatedAt.String(), updated.String())
	}
}

func TestSyncActivities(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	s := Server{DB: db}

	//service-1 is stuck with completed activities and creating
	svc1 := &model.Service{
		Name:      "service-1",
		Manifest:  "manifest",
		ProjectID: "project-id",
		Status:    model.DestroyedService,
	}
	err := db.Save(&svc1).Error
	if err != nil {
		t.Fatalf("error when saving service-1: %s", err.Error())
	}

	//activity is in progress more than 15 min and it is not generating logs
	a1 := &model.Activity{
		ServiceID: svc1.ID,
		Status:    model.InProgress,
		Type:      model.Destroyed,
	}
	err = db.Save(&a1).Error
	if err != nil {
		t.Fatalf("error when saving activity: %s", err.Error())
	}
	err = db.Model(&a1).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update activity:  %s", err.Error())
	}
	l1 := &model.ActivityLog{
		ActivityID: a1.ID,
		Log:        "this line",
	}
	err = db.Save(&l1).Error
	if err != nil {
		t.Fatalf("error when saving log: %s", err.Error())
	}
	err = db.Model(&l1).UpdateColumn("created_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update log-2:  %s", err.Error())
	}

	s.syncActivities()

	err = s.DB.Where("id = ?", a1.ID).First(&a1).Error
	if err != nil {
		t.Fatalf("failed to get activity-1:  %s", err.Error())
	}
	if a1.Status != model.Failed {
		t.Fatalf("Activity wrong status after sync: %s", a1.Status)
	}

	err = s.DB.Where("id = ?", svc1.ID).First(&svc1).Error
	if err != nil {
		t.Fatalf("failed to get service-1:  %s", err.Error())
	}
	if svc1.Status != model.FailedService {
		t.Fatalf("Service-1 wrong status after sync: %s", svc1.Status)
	}

	var logs []model.ActivityLog
	err = s.DB.Where("activity_id = ?", a1.ID).Find(&logs).Error
	if err != nil {
		t.Fatalf("failed to get logs:  %s", err.Error())
	}
	if len(logs) != 2 {
		t.Fatalf("Wrong number of logs in activity-1: %v", logs)
	}
	if logs[0].Log != "Internal Server Error: activity timedout" && logs[1].Log != "Internal Server Error: activity timedout" {
		t.Fatalf("Wrong error log in activity-1: %v", logs)
	}
}

func TestStuckServicies(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	s := Server{DB: db}

	//service-1 is stuck with completed activities and creating
	svc1 := &model.Service{
		Name:      "service-1",
		Manifest:  "manifest",
		ProjectID: "project-id",
		Status:    model.CreatingService,
	}
	err := db.Save(&svc1).Error
	if err != nil {
		t.Fatalf("error when saving service-1: %s", err.Error())
	}
	err = db.Model(&svc1).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-1:  %s", err.Error())
	}

	//service-2 is stuck with completed activities and deploying
	svc2 := &model.Service{
		Name:      "service-2",
		Manifest:  "manifest",
		ProjectID: "project-id",
		Status:    model.DeployingService,
	}
	err = db.Save(&svc2).Error
	if err != nil {
		t.Fatalf("error when saving service-2: %s", err.Error())
	}
	err = db.Model(&svc2).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-2:  %s", err.Error())
	}

	//service-3 is stuck with completed activities and destroying
	svc3 := &model.Service{
		Name:      "service-3",
		Manifest:  "manifest",
		ProjectID: "project-id",
		Status:    model.DestroyingService,
	}
	err = db.Save(&svc3).Error
	if err != nil {
		t.Fatalf("error when saving service-3: %s", err.Error())
	}
	err = db.Model(&svc3).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-3:  %s", err.Error())
	}

	//service-4 is not stuck due to "inprogress" activity
	svc4 := &model.Service{
		Name:      "service-4",
		Manifest:  "manifest",
		ProjectID: "project-id",
		Status:    model.DestroyingService,
	}
	err = db.Save(&svc4).Error
	if err != nil {
		t.Fatalf("error when saving service-4: %s", err.Error())
	}
	a1 := &model.Activity{
		ServiceID: svc4.ID,
		Status:    model.InProgress,
		Type:      model.Deployed,
	}
	err = db.Save(&a1).Error
	if err != nil {
		t.Fatalf("error when saving activity-1: %s", err.Error())
	}
	err = db.Model(&a1).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update activity-1:  %s", err.Error())
	}

	//service-5 is not stuck with recently complete activity
	svc5 := &model.Service{
		Name:      "service-5",
		Manifest:  "manifest",
		ProjectID: "project-id",
		Status:    model.DestroyingService,
	}
	err = db.Save(&svc5).Error
	if err != nil {
		t.Fatalf("error when saving service-5: %s", err.Error())
	}
	a2 := &model.Activity{
		ServiceID: svc5.ID,
		Status:    model.Completed,
		Type:      model.Deployed,
	}
	err = db.Save(&a2).Error
	if err != nil {
		t.Fatalf("error when saving activity-2: %s", err.Error())
	}

	services := s.stuckServices()
	if len(services) != 3 {
		t.Fatalf("Wrong stuck servicies: %v", services)
	}
	for i := 0; i < 3; i++ {
		if services[i].ID != svc1.ID && services[i].ID != svc2.ID && services[i].ID != svc3.ID {
			t.Fatalf("Wrong stuck service: %v", services[i])
		}
	}
}

func TestSyncServices(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	s := Server{DB: db}

	//service-1 is stuck with completed activities and creating
	svc1 := &model.Service{
		Name:      "service-1",
		Manifest:  "manifest",
		ProjectID: "project-id",
		Status:    model.DeployingService,
	}
	err := db.Save(&svc1).Error
	if err != nil {
		t.Fatalf("error when saving service-1: %s", err.Error())
	}
	err = db.Model(&svc1).UpdateColumn("updated_at", time.Now().Add(-20*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update service-1:  %s", err.Error())
	}

	//activity-1 is completed more than 1 min
	a1 := &model.Activity{
		ServiceID: svc1.ID,
		Status:    model.Completed,
		Type:      model.Deployed,
	}
	err = db.Save(&a1).Error
	if err != nil {
		t.Fatalf("error when saving activity-1: %s", err.Error())
	}
	err = db.Model(&a1).UpdateColumn("updated_at", time.Now().Add(-10*time.Minute).UTC()).Error
	if err != nil {
		t.Fatalf("failed to update activity-1:  %s", err.Error())
	}

	s.syncServices()

	err = s.DB.Where("id = ?", svc1.ID).First(&svc1).Error
	if err != nil {
		t.Fatalf("failed to get service-1:  %s", err.Error())
	}
	if svc1.Status != model.FailedService {
		t.Fatalf("Service-1 wrong status after sync: %s", svc1.Status)
	}
}
