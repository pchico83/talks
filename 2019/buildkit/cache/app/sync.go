package app

import (
	"fmt"
	"time"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/pkg/errors"
)

func (s *Server) sync() {
	for {
		s.syncActivities()
		s.syncServices()
		time.Sleep(60 * time.Second)
	}
}

func (s *Server) syncActivities() {
	activities := s.stuckActivities()
	for _, a := range activities {
		logger.Error(fmt.Errorf("activity-%s is stuck, it has been in progress since %s", a.ID, a.UpdatedAt.String()))
		s.addLog(a.ID, "Internal Server Error: activity timedout")
		r := s.DB.Model(&a).Update("status", model.Failed)
		if r.Error != nil {
			logger.Error(errors.Wrapf(r.Error, "failed to timeout stuck activity-%s", a.ID))
			continue
		}
		var svc model.Service
		r = s.DB.Where("id = ?", a.ServiceID).First(&svc)
		if r.Error != nil {
			logger.Error(errors.Wrapf(r.Error, "failed to get service-%s", a.ServiceID))
			continue
		}
		r = s.DB.Model(&svc).Update("status", model.Failed)
		if r.Error != nil {
			logger.Error(errors.Wrapf(r.Error, "failed to timeout stuck service-%s", svc.ID))
			continue
		}
	}
}

func (s *Server) stuckActivities() []model.Activity {
	stuckPeriod := time.Now().Add(-15 * time.Minute).UTC()
	var candidates []model.Activity
	var result []model.Activity
	r := s.DB.Where("updated_at < ? AND status =  ?", stuckPeriod, model.InProgress).Find(&candidates)
	if r.Error != nil {
		logger.Error(errors.Wrap(r.Error, "failed to get stuck activities"))
		return result
	}
	for _, a := range candidates {
		var count int
		r = s.DB.Model(&model.ActivityLog{}).Where("activity_id = ? AND created_at > ?", a.ID, stuckPeriod).Count(&count)
		if r.Error != nil {
			logger.Error(errors.Wrap(r.Error, "failed to get stuck activity logs"))
			continue
		}
		if count == 0 {
			result = append(result, a)
		} else {
			r = s.DB.Model(&a).UpdateColumn("updated_at", time.Now().UTC())
			if r != nil {
				logger.Error(errors.Wrapf(r.Error, "failed to refresh activity-%s", a.ID))
				continue
			}
		}
	}
	return result
}

func (s *Server) syncServices() {
	services := s.stuckServices()
	for _, svc := range services {
		logger.Error(fmt.Errorf("service-%s is stuck, marking it as failed", svc.ID))
		r := s.DB.Model(&svc).Update("status", model.FailedService)
		if r.Error != nil {
			logger.Error(errors.Wrapf(r.Error, "failed to timeout service-%s", svc.ID))
		}
	}
}

func (s *Server) stuckServices() []model.Service {
	serviceStuckPeriod := time.Now().Add(-15 * time.Minute).UTC()
	activityStuckPeriod := time.Now().Add(-5 * time.Minute).UTC()
	var candidates []model.Service
	var result []model.Service
	r := s.DB.Where("status in (?) AND updated_at < ?", []string{"creating", "deploying", "destroying"}, serviceStuckPeriod).Find(&candidates)
	if r.Error != nil {
		logger.Error(errors.Wrap(r.Error, "failed to sync services"))
		return result
	}
	if len(candidates) > 0 {
		logger.Info("found %d stuck service candidates", len(candidates))
	}
	for _, svc := range candidates {
		var count int
		r := s.DB.Model(&model.Activity{}).Where("service_id = ? AND (status = ? AND updated_at > ?)", svc.ID, model.InProgress, activityStuckPeriod).Count(&count)
		if r.Error != nil {
			logger.Error(errors.Wrapf(r.Error, "failed to get info from service-%s", svc.ID))
			continue
		}
		if count == 0 {
			result = append(result, svc)
		}
	}
	return result
}
