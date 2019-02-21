package app

import "testing"

func Test_isInternalUser(t *testing.T) {
	type args struct {
		email string
	}
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"okteto", "ramiro@okteto.com", true},
		{"gmail", "ramiro@gmail.com", false},
		{"gmail", "rberrelleza@gmail.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInternalUser(tt.email); got != tt.want {
				t.Errorf("isInternalUser() = %v, want %v", got, tt.want)
			}
		})
	}
}
