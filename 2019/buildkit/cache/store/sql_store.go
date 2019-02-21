package store

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/database"

	"bitbucket.org/okteto/okteto/backend/migrations"
	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/golang-migrate/migrate"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	// Used by migrate
	_ "github.com/golang-migrate/migrate/source/file"
)

const (
	currentSchema = 10
)

// InitSQLStore creates the tables and migrates if needed
func InitSQLStore(db *gorm.DB, driver database.Driver, migrationsPath string) error {
	err := createTables(db)
	if err != nil {
		return err
	}

	prevVersion, err := migrateSchema(driver, migrationsPath)

	if prevVersion == 8 {
		if err := migrations.MigrateServiceCreatedBy(db); err != nil {
			return err
		}
		prevVersion = 9
	}

	return err
}

func createTables(db *gorm.DB) error {
	result := db.AutoMigrate(
		&model.Service{},
		&model.User{},
		&model.Activity{},
		&model.ActivityLog{},
		&model.Project{},
		&model.ProjectACL{},
		&model.GHRepoLink{},
		&model.GHInstallation{})

	if result.Error != nil {
		return errors.Wrap(result.Error, "Failed to create the tables")
	}

	for _, tbl := range []string{"services", "users", "activities", "activity_logs"} {
		if !db.HasTable(tbl) {
			return errors.Wrap(result.Error, fmt.Sprintf("Table %s is missing", tbl))
		}
	}

	return nil
}

func migrateSchema(driver database.Driver, migrationsPath string) (uint, error) {
	log.Printf("reading migrations from: %s", migrationsPath)
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "database", driver)

	if err != nil {
		return 0, errors.Wrap(err, "Failed to create migration object")
	}

	prevVersion, _, _ := m.Version()

	err = m.Migrate(currentSchema)
	if err != migrate.ErrNoChange {
		return 0, errors.Wrap(err, "Failed to migrate data")
	}

	return prevVersion, nil
}
