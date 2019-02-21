package store

import (
	"testing"

	"bitbucket.org/okteto/okteto/backend/config"
	"github.com/golang-migrate/migrate/database/postgres"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
)

func TestMigrationIsSuccesful(t *testing.T) {
	viper.AddConfigPath("../config")
	config.LoadConfig()

	dialect, dbinfo := config.GetDBConnectionString()
	db, err := gorm.Open(dialect, dbinfo)
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	driver, err := postgres.WithInstance(db.DB(), &postgres.Config{})
	if err != nil {
		t.Fatal(err)
	}

	err = InitSQLStore(db, driver, "file://../migrations")
	if err != nil {
		t.Fatalf("couldn't run the migrations: %s", err.Error())
	}

	result := db.Raw("Select version, dirty from schema_migrations")
	rows, err := result.Rows()
	if err != nil {
		t.Fatalf("couldn't get the schema_versions data: %s", err.Error())
	}

	defer rows.Close()
	for rows.Next() {
		var version int
		var dirty bool
		err := rows.Scan(&version, &dirty)
		if err != nil {
			t.Fatalf(err.Error())
		}

		if dirty {
			t.Errorf("schema_migration is dirty")
		}

		if version != currentSchema {
			t.Errorf("schema_migration is not in the expected version: %d", version)
		}
	}
}
