package providers

import (
	"testing"

	"bitbucket.org/okteto/okteto/backend/model"
)

func TestFindDevContainer(t *testing.T) {
	// service with single container no dev configuration
	// service with two containers no dev configuration
	// service with two containers and one has dev configuration
	tests := []struct {
		name       string
		containers map[string]*model.Container
		expected   string
	}{
		{
			name:     "no dev configuration",
			expected: "ubuntu",
			containers: map[string]*model.Container{
				"ubuntu": &model.Container{Image: "ubuntu"},
			},
		},
		{
			name:     "multiple containers no dev configuration",
			expected: "python",
			containers: map[string]*model.Container{
				"ubuntu": &model.Container{Image: "ubuntu"},
				"python": &model.Container{Image: "python"},
			},
		},
		{
			name:     "multiple containers dev configuration",
			expected: "ruby",
			containers: map[string]*model.Container{
				"ubuntu": &model.Container{Image: "ubuntu"},
				"ruby":   &model.Container{Image: "ruby", Development: &model.Development{Image: "ruby-dev"}},
				"python": &model.Container{Image: "python"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &model.Service{
				Containers: tt.containers,
			}

			c := findDevContainer(s)
			if c.Image != tt.expected {
				t.Errorf("expected %s but got %s", tt.expected, c.Image)
			}
		})
	}

}
