package app

import (
	"testing"
)

func TestNewMail(t *testing.T) {
	type args struct {
		apiKey     string
		mailDomain string
		fromEmail  string
	}
	tests := []struct {
		name string
		args args
	}{
		{"validate templates", args{"key-123456", "example.com", "hello@okteto.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMail(tt.args.fromEmail, &NoopMail{})
			if m == nil {
				t.Errorf("mailgun instance not created")
			}

			if m.projectInvite == nil {
				t.Errorf("projectInvite template not created")
			}

			if m.projectInviteHTML == nil {
				t.Errorf("projectInviteHTML template not created")
			}

			if m.userInvite == nil {
				t.Errorf("userInvite template not created")
			}

			if m.userInviteHTML == nil {
				t.Errorf("userInviteHTML template not created")
			}
		})
	}
}
