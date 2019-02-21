package store

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"

	// for tests to run in memory with real SQL
	_ "github.com/mattn/go-sqlite3"
)

// NewMemoryStore returns a gorm.DB instance configured to use SQLite3 in memory
func NewMemoryStore() *gorm.DB {
	db := newMemoryStore()
	err := createTables(db)
	if err != nil {
		log.Fatalf("Failed to create tables: %s", err.Error())
	}

	return db
}

func newMemoryStore() *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("Failed to open sqlitedb: %s", err.Error())
	}

	gorm.NowFunc = func() time.Time {
		return time.Now().UTC()
	}

	db.DB().SetMaxOpenConns(1)
	return db
}
