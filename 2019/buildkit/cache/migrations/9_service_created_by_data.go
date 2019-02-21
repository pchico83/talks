package migrations

import (
	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/jinzhu/gorm"
)

//MigrateServiceCreatedBy migrates data for migration 8
func MigrateServiceCreatedBy(db *gorm.DB) error {
	var services []model.Service

	result := db.Find(&services)
	if result.Error != nil {
		return result.Error
	}

	for _, svc := range services {
		var activity model.Activity
		result = db.Where("service_id = ?", svc.ID).First(&activity)
		if result.Error != nil {
			return result.Error
		}
		result = db.Model(&svc).Update("created_by", activity.ActorID)
		if result.Error != nil {
			return result.Error
		}

		var project model.Project
		result = db.Where("id = ?", svc.ProjectID).First(&project)
		if result.Error != nil {
			return result.Error
		}

		settings, err := model.ParseProjectSettings(project.Settings)
		if err != nil {
			return err
		}
		result = db.Model(&svc).Update("is_demo", settings.Provider.Type == model.Demo)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}
