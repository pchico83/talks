package model

import (
	"testing"
	"time"
)

func newActivity(t time.Time, at ActivityType) *Activity {
	a := &Activity{}
	a.UpdatedAt = t
	a.Type = at
	return a
}

func TestIsOlder(t *testing.T) {

	older, _ := time.Parse(time.RFC3339, "2018-07-02T15:04:05+07:00")
	newer, _ := time.Parse(time.RFC3339, "2018-07-22T18:01:06+07:00")

	var tables = []struct {
		a       *Activity
		b       *Activity
		isOlder bool
	}{
		{newActivity(older, Created), newActivity(newer, Deployed), true},
		{newActivity(newer, Created), newActivity(newer, Created), false},
		{newActivity(newer, Created), newActivity(newer, Deployed), true},
		{newActivity(newer, Created), newActivity(newer, Destroyed), true},
		{newActivity(newer, Deployed), newActivity(newer, Destroyed), true},
		{newActivity(newer, Destroyed), newActivity(newer, Deployed), false},
	}

	for _, tt := range tables {
		isOlder := tt.a.IsOlder(tt.b)
		if isOlder != tt.isOlder {
			t.Errorf("expected %t, got %t \n %+v \n %+v", tt.isOlder, isOlder, tt.a, tt.b)
		}
	}
}
