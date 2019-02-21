package app

import (
	"fmt"
	"time"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/pkg/errors"
)

func (s *Server) cleanExpiredServices() {
	for {
		s.clean()
		time.Sleep(10 * 60 * time.Second)
	}
}

func (s *Server) clean() {
	var u model.User
	r := s.DB.Where("email = ?", "bot@okteto.com").First(&u)
	if r.Error != nil {
		logger.Error(errors.Wrap(r.Error, "failed to get bot user"))
		return
	}

	services := s.expiredServices()
	for _, svc := range services {
		var p model.Project
		result := s.DB.Where("id = ?", svc.ProjectID).First(&p)
		if result.Error != nil {
			logger.Error(errors.Wrapf(result.Error, "failed to get expired service project-%s", svc.ProjectID))
			continue
		}
		settings, err := model.ParseProjectSettings(p.Settings)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to parse provider settings project-%s", svc.ProjectID))
			continue
		}
		if settings.Provider.Type != model.Demo {
			logger.Error(fmt.Errorf("failed to remove expired service-%s in non demo provider", svc.ID))
			s.DB.Model(&svc).Update("is_demo", false)
			continue
		}
		p.LoadedSettings = settings
		if err := s.DeleteService(&p, svc.ID, &u); err != nil {
			logger.Error(errors.Wrapf(err, "failed to delete expired service-%s", svc.ID))
		}
	}
}

func (s *Server) expiredServices() []model.Service {
	expiredPeriod := time.Now().Add(-60 * time.Minute).UTC()
	var services []model.Service
	r := s.DB.Where(
		"status in (?) AND created_at < ? AND is_demo = ?",
		[]string{"created", "failed", "deployed"}, expiredPeriod, true).Find(&services)
	if r.Error != nil {
		logger.Error(errors.Wrap(r.Error, "failed to get expired demo services"))
		return services
	}
	return services
}
