package app

import (
	"fmt"
	"log"
	"strings"
	"time"

	"bitbucket.org/okteto/okteto/backend/logger"

	"bitbucket.org/okteto/okteto/backend/config"
	"bitbucket.org/okteto/okteto/backend/model"
	K8Service "bitbucket.org/okteto/okteto/backend/providers/k8/service"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

const (
	maxDemoServices = 5
)

// CreateService validates the compose and the device, and if valid, saves it to the
// DB.
func (s *Server) CreateService(project *model.Project, service *model.Service, user *model.User) (*model.Service, *model.AppError) {
	if !strings.HasSuffix(user.Email, "okteto.com") {
		var count int
		r := s.DB.Model(&model.Service{}).Where("created_by = ? AND is_demo = ?", service.CreatedBy, true).Count(&count)
		if r.Error != nil {
			return nil, &model.AppError{Status: 500, Message: r.Error.Error()}
		}
		if count >= maxDemoServices {
			return nil, &model.AppError{
				Status:  429,
				Code:    model.MissingProjectSettings,
				Message: fmt.Sprintf("You have already created %d services using the demo provider", maxDemoServices)}
		}
	}
	service.ProjectID = project.ID

	m, appErr := model.ParseEncodedManifest(service.Manifest)
	if appErr != nil {
		return nil, appErr
	}

	activity := model.Activity{
		ActorID:   user.ID,
		ServiceID: service.ID,
		Type:      model.Created,
		Status:    model.Completed,
	}

	service.Name = m.Name
	service.Status = service.CalculateStatus(&activity)
	service.Activities = []model.Activity{activity}
	result := s.DB.Create(service)

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "service_unique_name") {
			return nil, &model.AppError{Status: 400, Code: model.UniqueName, Message: result.Error.Error()}
		}

		return nil, &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	go s.newServiceNotification(user)
	return service, nil
}

func (s *Server) getService(project *model.Project, serviceID string) (*model.Service, *model.AppError) {
	var service model.Service
	result := s.DB.Where("id = ? AND project_id = ?", serviceID, project.ID).First(&service)
	err := s.validateServiceResult(&service, result)
	if err != nil {
		return nil, err
	}

	return &service, nil
}

// GetServiceByID returns the service based on the ID. This doesn't validate  permissions or ownership.
func (s *Server) GetServiceByID(serviceID string) (*model.Service, *model.AppError) {
	if serviceID == "" {
		return nil, &model.AppError{Status: 400, Code: model.MissingManifest}
	}

	var service model.Service
	result := s.DB.Where("id = ?", serviceID).First(&service)
	err := s.validateServiceResult(&service, result)
	if err != nil {
		return nil, err
	}

	return &service, nil
}

//GetServiceAndActivities returns a service with all its activities
func (s *Server) GetServiceAndActivities(project *model.Project, serviceID string) (*model.Service, *model.AppError) {
	var service model.Service
	result := s.DB.Where("id = ? AND project_id = ?", serviceID, project.ID).First(&service)

	appErr := s.validateServiceResult(&service, result)
	if appErr != nil {
		return nil, appErr
	}

	result = s.DB.Raw("select a.id, a.created_at, a.updated_at, a.type, a.status, u.email as actor_email from activities as a LEFT JOIN users as u on a.actor_id = u.id WHERE service_id = ? ORDER BY a.updated_at ASC", serviceID)
	rows, err := result.Rows()

	if err != nil {
		return nil, &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	defer rows.Close()

	var activities []model.Activity
	for rows.Next() {
		var activity model.Activity
		err := rows.Scan(&activity.ID, &activity.CreatedAt, &activity.UpdatedAt, &activity.Type, &activity.Status, &activity.ActorEmail)
		if err != nil {
			return nil, &model.AppError{Status: 500, Code: model.InternalServerError, Message: err.Error()}
		}

		activities = append(activities, activity)
	}

	service.Activities = activities
	s.buildLinks(&service, project)
	return &service, nil

}

func (s *Server) validateServiceResult(service *model.Service, result *gorm.DB) *model.AppError {
	if result.RecordNotFound() {
		return &model.AppError{Status: 404, Code: model.EntityNotFound, Message: result.Error.Error()}
	}

	if result.Error != nil {
		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	return nil
}

// GetServices gets all services that belong to the same project
func (s *Server) GetServices(projectID string, includeDeleted bool) ([]model.Service, *model.AppError) {
	var services []model.Service

	var result = s.DB

	if includeDeleted {
		result = result.Where("project_id = ?", projectID).Find(&services)
	} else {
		deletedThreshold := time.Now().UTC().Add(-1 * time.Hour)
		result = result.Where(
			"project_id = ? AND (status != ? OR (status = ? AND updated_at > ?))",
			projectID, model.DestroyedService, model.DestroyedService, deletedThreshold).Find(&services)
	}

	if result.RecordNotFound() {
		return nil, &model.AppError{Status: 404, Code: model.EntityNotFound, Message: result.Error.Error()}
	}

	if result.Error != nil {
		return nil, &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	for i, sv := range services {
		services[i].Links = buildServiceLinks(sv.ProjectID, sv.ID)
	}

	return services, nil
}

func (s *Server) getActiveServicesCount(projectID string) (int, error) {
	var count int
	result := s.DB.Model(&model.Service{}).Where("project_id = ? AND status != ?", projectID, model.DestroyedService).Count(&count)
	return count, result.Error
}

// UpdateServiceActivity updates the status of the activity
func (s *Server) updateServiceActivity(project *model.Project, serviceID string, activityID string, newStatus model.ActivityStatus, dns *string) error {

	var activity model.Activity
	result := s.DB.Where("id = ?", activityID).First(&activity)
	if result.Error != nil {
		return errors.Wrap(result.Error, fmt.Sprintf("couldn't find activity-%s", activityID))
	}

	if activity.Status != model.InProgress {
		logger.Error(fmt.Errorf("invalid-service-state: activity-%s is not in progress", activityID))
	}

	result = s.DB.Model(&activity).Update("status", newStatus)

	if result.Error != nil {
		return result.Error
	}

	service := model.Service{}
	service.ID = serviceID
	service.Status = service.CalculateStatus(&activity)

	if dns != nil {
		service.DNS = *dns
	}

	result = s.DB.Model(&service).Updates(map[string]interface{}{"status": service.Status, "dns": service.DNS})
	if result.Error != nil {
		return errors.Wrap(result.Error, fmt.Sprintf("failed to update service-%s is in a bad state", serviceID))
	}

	return nil
}

// StartService starts a service, using the latest data from the DB
func (s *Server) StartService(project *model.Project, serviceID string, user *model.User) *model.AppError {
	service, err := s.getService(project, serviceID)
	if err != nil {
		return err
	}

	if !service.CanDeploy() {
		return &model.AppError{Status: 400, Code: model.InvalidServiceStatus}
	}

	if service.Manifest == "" {
		return &model.AppError{Status: 400, Code: model.MissingManifest}
	}

	activity := model.Activity{
		ActorID:   user.ID,
		ServiceID: serviceID,
		Type:      model.Deployed,
		Status:    model.InProgress,
	}

	result := s.DB.Create(&activity)
	if result.Error != nil {
		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	result = s.DB.Model(service).Updates(map[string]interface{}{"status": service.CalculateStatus(&activity), "dev": false})
	if result.Error != nil {
		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	go s.deployedServiceNotification(user)

	// launch deploy in a goroutine
	go func(p *model.Project, d *model.Service, activityID string) {
		s.pendingOperations.Add(1)
		defer s.pendingOperations.Done()
		err := s.deploy(d, p, activityID)

		var activityStatus = model.Completed
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to start service-%s activity-%s", d.ID, activityID))
			s.addLog(activityID, err.Error())
			activityStatus = model.Failed

		} else {
			log.Printf("started service-%s activity-%s", d.ID, activityID)
			activityStatus = model.Completed
		}

		dns := s.buildProjectDNS(&p.DNSName, p.LoadedSettings)
		err = s.updateServiceActivity(p, d.ID, activityID, activityStatus, dns)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to update the service-%s activity-%s after the deploy operation was %s", d.ID, activityID, activityStatus))
		}
	}(project, service, activity.ID)
	return nil
}

// EnableDevMode turns a service into developer mode
func (s *Server) EnableDevMode(project *model.Project, serviceID string, user *model.User) *model.AppError {
	service, err := s.getService(project, serviceID)
	if err != nil {
		return err
	}

	if service.Dev {
		return nil
	}

	if !service.CanEnableDev() {
		return &model.AppError{Status: 400, Code: model.InvalidServiceStatus}
	}

	activity := model.Activity{
		ActorID:   user.ID,
		ServiceID: serviceID,
		Type:      model.DevDeployed,
		Status:    model.InProgress,
	}

	result := s.DB.Create(&activity)
	if result.Error != nil {
		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	result = s.DB.Model(service).Updates(model.Service{Status: service.CalculateStatus(&activity), Dev: true})
	if result.Error != nil {
		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	go s.devDeployedServiceNotification(user)

	go func(p *model.Project, d *model.Service, activityID string) {
		s.pendingOperations.Add(1)
		defer s.pendingOperations.Done()
		err := s.devDeploy(d, p, activityID)

		var activityStatus = model.Completed
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to start dev mode service-%s activity-%s", d.ID, activityID))
			s.addLog(activityID, err.Error())
			activityStatus = model.Failed

		} else {
			log.Printf("dev mode enabled for service-%s activity-%s", d.ID, activityID)
			activityStatus = model.Completed
		}

		dns := s.buildProjectDNS(&p.DNSName, p.LoadedSettings)
		err = s.updateServiceActivity(p, d.ID, activityID, activityStatus, dns)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to update the service-%s activity-%s after the dev mode operation was %s", d.ID, activityID, activityStatus))
		}
	}(project, service, activity.ID)
	return nil
}

// DeleteService deletes a service from the DB based on the ID, or an error if not found
func (s *Server) DeleteService(project *model.Project, serviceID string, user *model.User) *model.AppError {
	service, appErr := s.getService(project, serviceID)
	if appErr != nil {
		log.Printf("failed to find service-%s after deletion: %s", serviceID, appErr.Message)
		return appErr
	}

	if service.IsDestroyed() {
		log.Printf("can't destroy, service-%s is already %s", service.ID, service.Status)
		return &model.AppError{Status: 400, Code: model.InvalidServiceStatus}
	}

	if !service.CanDestroy() {
		log.Printf("can't destroy, service-%s is %s", service.ID, service.Status)
		return &model.AppError{Status: 400, Code: model.InvalidServiceStatus}
	}

	activity := model.Activity{
		ActorID:   user.ID,
		ServiceID: serviceID,
		Type:      model.Destroyed,
		Status:    model.InProgress,
	}

	result := s.DB.Create(&activity)
	if result.Error != nil {
		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	result = s.DB.Model(service).Updates(map[string]interface{}{"status": service.CalculateStatus(&activity), "dns": nil})
	if result.Error != nil {
		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	// launch destroy in a goroutine
	go func(p *model.Project, d *model.Service, activityID string) {
		s.pendingOperations.Add(1)
		defer s.pendingOperations.Done()

		err := s.destroy(d, p, activityID)

		var activityStatus = model.Completed
		if err != nil {
			log.Printf("failed to delete service-%s: %s", d.ID, err.Error())
			s.addLog(activityID, err.Error())
			activityStatus = model.Failed
		}

		err = s.updateServiceActivity(p, d.ID, activityID, activityStatus, nil)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to update the service-%s activity-%s after the destroy operation was %s", d.ID, activityID, activityStatus))
		} else {
			log.Printf("deleted service-%s activity-%s", d.ID, activityID)
		}
	}(project, service, activity.ID)

	return nil
}

// UpdateManifest updates the service's manifest.
func (s *Server) UpdateManifest(projectID, serviceID, manifest string, actorID string, updateLog string) *model.AppError {

	m, appErr := model.ParseEncodedManifest(manifest)
	if appErr != nil {
		return appErr
	}

	svc := model.Service{Model: model.Model{ID: serviceID}}
	result := s.DB.Model(&svc).
		Where("id = ?", serviceID).Where("project_id = ?", projectID).
		Updates(model.Service{Manifest: manifest, Name: m.Name})

	if result.Error != nil {
		// handle service_unique_name
		if strings.Contains(result.Error.Error(), "service_unique_name") {
			return &model.AppError{Status: 400, Code: model.UniqueName, Message: result.Error.Error()}
		}

		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	activity := model.Activity{
		ActorID:   actorID,
		ServiceID: serviceID,
		Type:      model.Updated,
		Status:    model.Completed,
	}

	result = s.DB.Create(&activity)
	if result.Error != nil {
		return &model.AppError{Status: 500, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	s.addLog(activity.ID, updateLog)
	return nil
}

// GetActivityLogs returns the logs of a given identity, or a 404 if the logs don't exist
func (s *Server) GetActivityLogs(project *model.Project, serviceID string, activityID string) ([]model.ActivityLog, *model.AppError) {

	_, appErr := s.getService(project, serviceID)

	if appErr != nil {
		return nil, appErr
	}

	logs, err := s.getActivityLogs(activityID)

	if err != nil {
		return nil, &model.AppError{Status: 500, Code: model.InternalServerError, Message: err.Error()}
	}

	return logs, nil
}

func (s *Server) getActivityLogs(activityID string) ([]model.ActivityLog, error) {

	var logs []model.ActivityLog
	result := s.DB.Where("activity_id = ?", activityID).Order("updated_at ASC").Find(&logs)
	return logs, result.Error
}

func (s *Server) addLog(activityID string, logs string) error {
	log := model.ActivityLog{
		ActivityID: activityID,
		Log:        logs,
	}

	result := s.DB.Create(&log)

	if result.Error != nil {
		logger.Error(errors.Wrapf(result.Error, "Failed to save log for activity-%s", activityID))
	}

	return result.Error
}

func buildServiceLinks(projectID string, serviceID string) model.ServiceLinks {
	return model.ServiceLinks{
		Activities: fmt.Sprintf("%s/projects/%s/services/%s/activities", config.GetAPIURL(), projectID, serviceID),
		Self:       fmt.Sprintf("%s/projects/%s/services/%s", config.GetAPIURL(), projectID, serviceID),
	}
}

func (s *Server) buildServiceEndpoints(service *model.Service, project *model.Project, dns string) []string {
	p := project.LoadedSettings.Provider
	p.LoadDefaultCluster()
	endpoints := []string{}
	if dns == "" {
		return endpoints
	}
	e := &model.Environment{
		Name:        project.DNSName,
		ProjectName: project.Name,
		Provider:    p,
	}
	if p.IsIngress() {
		for _, hostname := range service.GetIngressHostnames(e) {
			if p.Ingress.TLS == nil {
				endpoints = append(endpoints, fmt.Sprintf("http://%s", hostname))
			} else {
				endpoints = append(endpoints, fmt.Sprintf("https://%s", hostname))
			}
		}
		return endpoints
	}
	var endpoint string
	if config.IsDNSConfigured() {
		endpoint = fmt.Sprintf("%s.%s", service.Name, dns)
	} else {
		target, err := K8Service.GetEndpoint(service, e)
		if err != nil {
			return endpoints
		}
		endpoint = target
	}
	for _, port := range service.GetLoadBalancerPorts() {
		if port == "443" {
			endpoints = append(endpoints, fmt.Sprintf("https://%s", endpoint))
		} else if port == "80" {
			endpoints = append(endpoints, fmt.Sprintf("http://%s", endpoint))
		} else {
			endpoints = append(endpoints, fmt.Sprintf("%s:%s", endpoint, port))
		}
	}
	return endpoints
}

func (s *Server) buildProjectDNS(projectDNSName *string, settings *model.ProjectSettings) *string {
	var dns string
	if settings == nil || settings.Provider == nil {
		log.Printf("error: provider doesn't have settings")
		dns = "unknown"
		return &dns
	}
	hostedZone := "example.com"
	if s.DNSProvider != nil {
		hostedZone = strings.TrimSuffix(s.DNSProvider.HostedZone, ".")
	}
	dns = fmt.Sprintf("%s.%s", *projectDNSName, hostedZone)
	return &dns
}

func (s *Server) buildLinks(service *model.Service, project *model.Project) {
	service.Links = buildServiceLinks(service.ProjectID, service.ID)

	m, appErr := buildService(service.ID, service.Manifest)
	if appErr != nil {
		logger.Error(errors.Wrap(appErr, "parse error when trying to build service links"))
		return
	}

	if project.LoadedSettings == nil {
		settings, _ := model.ParseProjectSettings(project.Settings)
		settings.Provider.LoadDefaultCluster()
		project.LoadedSettings = settings
	}

	service.Endpoints = s.buildServiceEndpoints(m, project, service.DNS)
	return

}
