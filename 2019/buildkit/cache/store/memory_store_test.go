package store

import "testing"

func TestMemoryStore(t *testing.T) {
	db := NewMemoryStore()
	defer db.Close()

	for _, tbl := range []string{"services", "users", "activities", "activity_logs"} {
		if !db.HasTable(tbl) {
			t.Errorf("%s wasn't created", tbl)
		}
	}
}
