package app

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/store"
	"github.com/pkg/errors"
)

func Test_validateACLs(t *testing.T) {
	tests := []struct {
		name     string
		users    map[string]*model.User
		acls     []model.ProjectACL
		existing []model.ProjectACL
		pending  []model.ProjectACL
	}{
		{
			name: "all exist",
			users: map[string]*model.User{
				"test-1@example.com": &model.User{Verified: true},
				"test-2@example.com": &model.User{Verified: true},
			},
			acls: []model.ProjectACL{
				model.ProjectACL{UserEmail: "test-1@example.com", ProjectID: "1"},
				model.ProjectACL{UserEmail: "test-2@example.com", ProjectID: "1"}},
			existing: []model.ProjectACL{
				model.ProjectACL{UserEmail: "test-1@example.com", ProjectID: "1"},
				model.ProjectACL{UserEmail: "test-2@example.com", ProjectID: "1"}},
			pending: []model.ProjectACL{},
		},
		{
			name: "some exist",
			users: map[string]*model.User{
				"test-1@example.com": &model.User{Verified: true},
				"test-2@example.com": &model.User{Verified: true},
				"test-3@example.com": &model.User{Verified: true},
			},
			acls: []model.ProjectACL{
				model.ProjectACL{UserEmail: "test-1@example.com", ProjectID: "1"},
				model.ProjectACL{UserEmail: "test-5@example.com", ProjectID: "1"}},
			existing: []model.ProjectACL{model.ProjectACL{UserEmail: "test-1@example.com", ProjectID: "1"}},
			pending:  []model.ProjectACL{model.ProjectACL{UserEmail: "test-5@example.com", ProjectID: "1"}},
		},
		{
			name: "some exist but not verified",
			users: map[string]*model.User{
				"test-1@example.com": &model.User{Verified: true},
				"test-2@example.com": &model.User{Verified: true},
				"test-3@example.com": &model.User{Verified: true},
				"test-5@example.com": &model.User{Verified: false},
			},
			acls: []model.ProjectACL{
				model.ProjectACL{UserEmail: "test-1@example.com", ProjectID: "1"},
				model.ProjectACL{UserEmail: "test-5@example.com", ProjectID: "1"},
			},
			existing: []model.ProjectACL{model.ProjectACL{UserEmail: "test-1@example.com", ProjectID: "1"}},
			pending:  []model.ProjectACL{model.ProjectACL{UserEmail: "test-5@example.com", ProjectID: "1"}},
		},
		{
			name: "none exist",
			users: map[string]*model.User{
				"test-1@example.com": &model.User{Verified: true},
				"test-2@example.com": &model.User{Verified: true},
				"test-3@example.com": &model.User{Verified: true},
			},
			acls: []model.ProjectACL{
				model.ProjectACL{UserEmail: "test-4@example.com", ProjectID: "1"},
				model.ProjectACL{UserEmail: "test-5@example.com", ProjectID: "1"}},
			existing: []model.ProjectACL{},
			pending:  []model.ProjectACL{model.ProjectACL{UserEmail: "test-4@example.com", ProjectID: "1"}, model.ProjectACL{UserEmail: "test-5@example.com", ProjectID: "1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := store.NewMemoryStore()
			defer db.Close()
			s := Server{DB: db}

			for k, v := range tt.users {
				_, err := s.createUser(k, v.Verified, "")
				if err != nil {
					t.Fatalf("%+v", err)
				}
			}

			existing, pending, appErr := s.validateACLs(tt.acls)

			if appErr != nil {
				t.Errorf("unexpected error: %+v", appErr)
				return
			}

			if !reflect.DeepEqual(tt.pending, pending) {
				t.Errorf("failed to calculate pending users. expecting: %+v, received: %+v", tt.pending, pending)
			}

			for i := range existing {
				if existing[i] != tt.existing[i] {
					t.Errorf("failed to calculate existing users. expecting: %s, received: %s", existing[i], tt.existing[i])
				}
			}

		})
	}
}

func TestCreateProject(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{Email: NewMail("hello@okteto.com", &NoopMail{}), DB: db}

	u, _ := s.createUser("test-1@example.com", true, "")
	u2, _ := s.createUser("test-2@example.com", true, "")

	p, appErr := s.CreateProject(&model.Project{Name: "testproject"}, u)
	if appErr != nil {
		t.Fatalf("unexpected error when creating a project: %+v", appErr)
	}

	p2, appErr := s.CreateProject(&model.Project{Name: "testproject"}, u2)
	if appErr != nil {
		t.Fatalf("unexpected error when creating a project: %+v", appErr)
	}

	getProjects, appErr := s.GetProjects(u)
	if appErr != nil {
		t.Fatalf("unexpected error when getting projects: %+v", appErr)
	}

	if len(getProjects) != 1 {
		t.Fatalf("got more projects than expected. Got %d instead of %d", len(getProjects), 1)
	}

	if getProjects[0].ID != p {
		t.Errorf("got the wrong project")
	}

	r, appErr := s.GetProject(p2, u.ID)
	if appErr == nil {
		t.Fatalf("User got a project that belonged to user 2: %+v", r)
	}

	if appErr.Status != http.StatusNotFound {
		t.Errorf("error didn't have status 404: %+v", appErr)
	}

	r, appErr = s.GetProject(p2, u2.ID)
	if appErr != nil {
		t.Fatalf("Didn't got a project: %+v", appErr)
	}

	if r.Role != model.ProjectRoleAdmin {
		t.Fatalf("Wrong role in project: %+v", r)
	}

	appErr = s.DeleteProject(p, false)
	if appErr != nil {
		t.Fatalf("Didn't delete a project: %+v", appErr)
	}

	projects, appErr := s.GetProjects(u)
	if appErr != nil {
		t.Fatalf("Didn't got projects: %+v", appErr)
	}

	if len(projects) != 0 {
		t.Fatalf("Result included deleted projects: %+v", projects)
	}

}

func TestCreateProjectWithPendingUsers(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	e := &testEmailSender{}
	s := Server{Email: NewMail("hello@okteto.com", e), DB: db}

	u, _ := s.createUser("test-1@example.com", true, "")
	projectID, appErr := s.CreateProject(&model.Project{Name: "testproject"}, u)
	if appErr != nil {
		t.Fatalf("unexpected error when creating a project: %+v", appErr)
	}

	project, appErr := s.GetProject(projectID, u.ID)
	if appErr != nil {
		t.Fatalf("unexpected error when getting a project: %+v", appErr)
	}

	returnedSettings, appErr := model.ParseProjectSettings(project.Settings)
	if appErr != nil {
		t.Fatalf("unexpected error: %+v \n %s", appErr, project.Settings)
	}

	if len(returnedSettings.Administrators) != 1 {
		t.Errorf("wrong settings returned for project: %+v", returnedSettings)
	}

	if returnedSettings.Administrators[0] != "test-1@example.com" {
		t.Errorf("wrong settings returned for project: %+v", returnedSettings)
	}

	if len(returnedSettings.Users) != 0 {
		t.Errorf("wrong settings returned for project: %+v", returnedSettings)
	}

	settings := model.ProjectSettings{
		Administrators: []string{"test-1@example.com"},
		Users:          []string{"test-2@example.com"},
		Provider: &model.Provider{
			Type: model.Demo,
		},
	}

	pendingUsers, appErr := s.UpdateProjectSettings(u, project, settings.Base64Encode())
	if appErr != nil {
		t.Fatalf("unexpected error when updating a project: %+v", appErr)
	}

	if len(pendingUsers) != 1 {
		t.Fatalf("failed to invite user to project: %+v", pendingUsers)
	}

	if pendingUsers[0].UserEmail != "test-2@example.com" {
		t.Fatalf("failed to invite user test-2 to project: %v", pendingUsers)
	}

	if e.sent != 0 {
		t.Error("email provider shoudn't be called for pending users")
	}
}

func TestShareProject(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	e := &testEmailSender{}
	s := Server{Email: NewMail("hello@okteto.com", e), DB: db}

	u, _ := s.createUser("test-1@example.com", true, "")
	u2, _ := s.createUser("test-2@example.com", true, "")

	projectID, appErr := s.CreateProject(&model.Project{Name: "testproject"}, u)
	if appErr != nil {
		t.Fatalf("unexpected error when creating a project: %+v", appErr)
	}

	p, appErr := s.GetProject(projectID, u2.ID)
	if appErr == nil {
		t.Errorf("second user could access project before sharing: %+v", appErr)
	}

	p, _ = s.GetProject(projectID, u.ID)

	settings := model.ProjectSettings{
		Administrators: []string{u.Email},
		Users:          []string{u2.Email},
		Provider: &model.Provider{
			Type: model.Demo,
		},
	}

	_, appErr = s.UpdateProjectSettings(u, p, settings.Base64Encode())

	if appErr != nil {
		t.Fatalf("unexpected error when updating a project: %+v", appErr)
	}

	p, appErr = s.GetProject(p.ID, u2.ID)
	if appErr != nil {
		t.Errorf("second user couldn't access project: %+v", appErr)
	}

	if p.Role != model.ProjectRoleUser {
		t.Errorf("user has the wrong role: %+v", p)
	}

	if e.sent == 0 {
		t.Fatalf("email provider wasn't called")
	}

	if e.email != u2.Email {
		t.Errorf("email provider didn't receive the corrrect email. Got: %s, expecting: %s", e.email, u2.Email)
	}

	settings = model.ProjectSettings{
		Administrators: []string{u.Email},
		Users:          []string{},
		Provider: &model.Provider{
			Type: model.Demo,
		},
	}

	p, _ = s.GetProject(projectID, u.ID)
	_, appErr = s.UpdateProjectSettings(u, p, settings.Base64Encode())
	if appErr != nil {
		t.Fatalf("updated failed: %+v", appErr)
	}

	p, appErr = s.GetProject(p.ID, u2.ID)
	if appErr == nil {
		t.Fatalf("second user accessed project after rights where removed: %+v", appErr)
	}
}

func TestShareProjectWithUnconfirmedUser(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	e := &testEmailSender{}
	s := Server{Email: NewMail("hello@okteto.com", e), DB: db}

	u, _ := s.createUser("test-1@example.com", true, "")
	u2, _ := s.createUser("test-2@example.com", false, "")

	projectID, appErr := s.CreateProject(&model.Project{Name: "testproject"}, u)
	if appErr != nil {
		t.Fatalf("unexpected error when creating a project: %+v", appErr)
	}

	p, _ := s.GetProject(projectID, u.ID)

	settings := model.ProjectSettings{
		Administrators: []string{u.Email},
		Users:          []string{u2.Email},
		Provider: &model.Provider{
			Type: model.Demo,
		},
	}

	pending, appErr := s.UpdateProjectSettings(u, p, settings.Base64Encode())

	if appErr != nil {
		t.Fatalf("unexpected error when updating a project: %+v", appErr)
	}

	_, appErr = s.GetProject(p.ID, u2.ID)
	if appErr == nil {
		t.Errorf("second user got access to the project: %+v", appErr)
	}

	if len(pending) == 0 {
		t.Error("unconfirmed user wasn't marked as pending")
	}

	if pending[0].UserEmail != u2.Email {
		t.Error("unconfirmed user wasn't returned as pending")
	}
}

func TestGetProjectsEmpty(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}

	u, _ := s.createUser("test-1@example.com", true, "")

	p, err := s.GetProjects(u)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if p == nil {
		t.Errorf("projects where nil instead of empty")
	}
}

func TestCreateProjectsSameName(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	u, _ := s.createUser("test-1@example.com", true, "")

	for i := 0; i < 30; i++ {
		_, appErr := s.CreateProject(&model.Project{Name: "testproject"}, u)
		if appErr != nil {
			t.Fatalf("unexpected error: %+v", appErr)
		}
	}

	p, appErr := s.GetProjects(u)
	if appErr != nil {
		t.Fatalf("unexpected error: %+v", appErr)
	}

	if len(p) != 30 {
		t.Fatalf("expecting %d projects but got %d", 30, len(p))
	}

	for i := 2; i < 30; i++ {
		expected := fmt.Sprintf("testproject-%d", i)
		if p[i].DNSName != expected {
			t.Fatalf("expecting %s got %s", expected, p[i].DNSName)
		}
	}
}
func Test_findInvitedUsers(t *testing.T) {
	type args struct {
		originalList []string
		updatedList  []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "empty list",
			args: args{originalList: []string{}, updatedList: []string{}},
			want: []string{},
		},
		{
			name: "no one invited",
			args: args{originalList: []string{"a@example.com"}, updatedList: []string{"a@example.com"}},
			want: []string{},
		},
		{
			name: "one invited",
			args: args{originalList: []string{"a@example.com"}, updatedList: []string{"a@example.com", "b@example.com"}},
			want: []string{"b@example.com"},
		},
		{
			name: "one invited different order",
			args: args{originalList: []string{"a@example.com"}, updatedList: []string{"b@example.com", "a@example.com"}},
			want: []string{"b@example.com"},
		},
		{
			name: "all new",
			args: args{originalList: []string{"a@example.com"}, updatedList: []string{"b@example.com", "c@example.com"}},
			want: []string{"b@example.com", "c@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findInvitedUsers(tt.args.originalList, tt.args.updatedList); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findInvitedUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_uniqueUsers(t *testing.T) {
	tests := []struct {
		name  string
		users []model.ProjectACL
		want  []model.ProjectACL
	}{
		{
			"all unique",
			[]model.ProjectACL{
				model.ProjectACL{UserEmail: "user1@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user2@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user3@example.com", Role: model.ProjectRoleAdmin},
			},
			[]model.ProjectACL{
				model.ProjectACL{UserEmail: "user1@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user2@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user3@example.com", Role: model.ProjectRoleAdmin},
			},
		},
		{
			"one duplicated, same role",
			[]model.ProjectACL{
				model.ProjectACL{UserEmail: "user1@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user2@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user1@example.com", Role: model.ProjectRoleAdmin},
			},
			[]model.ProjectACL{
				model.ProjectACL{UserEmail: "user1@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user2@example.com", Role: model.ProjectRoleAdmin},
			},
		},
		{
			"one duplicated, different role",
			[]model.ProjectACL{
				model.ProjectACL{UserEmail: "user1@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user2@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user1@example.com", Role: model.ProjectRoleUser},
			},
			[]model.ProjectACL{
				model.ProjectACL{UserEmail: "user1@example.com", Role: model.ProjectRoleAdmin},
				model.ProjectACL{UserEmail: "user2@example.com", Role: model.ProjectRoleAdmin},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueUsers(tt.users)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("uniqueUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkForUniqueDNSError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "postgres error", err: errors.New("pq: duplicate key value violates unique constraint \"unique_dns_name\""), want: true},
		{name: "sqllite error", err: errors.New("UNIQUE constraint failed: projects.dns_name"), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkForUniqueDNSError(tt.err); got != tt.want {
				t.Errorf("checkForUniqueDNSError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateGithubScope(t *testing.T) {
	type args struct {
		existing   *model.Github
		new        *model.Github
		loggedUser *model.User
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 *model.AppError
	}{
		{
			name:  "no github settings",
			want:  "",
			want1: nil,
			args:  args{},
		},
		{
			name:  "no old settings",
			want:  "12345",
			want1: nil,
			args: args{
				existing:   nil,
				new:        &model.Github{LinkedBy: "user1@example.com"},
				loggedUser: &model.User{Email: "user1@example.com", GHInstallation: &model.GHInstallation{Model: model.Model{ID: "12345"}}},
			},
		},
		{
			name:  "valid new settings",
			want:  "12345",
			want1: nil,
			args: args{
				existing:   &model.Github{LinkedBy: "user2@example.com"},
				new:        &model.Github{LinkedBy: "user1@example.com"},
				loggedUser: &model.User{Email: "user1@example.com", GHInstallation: &model.GHInstallation{Model: model.Model{ID: "12345"}}},
			},
		},
		{
			name:  "valid new settings removing",
			want:  "",
			want1: nil,
			args: args{
				existing:   &model.Github{LinkedBy: "user2@example.com"},
				new:        nil,
				loggedUser: &model.User{Email: "user1@example.com", GHInstallation: &model.GHInstallation{Model: model.Model{ID: "12345"}}},
			},
		},
		{
			name: "invalid github scope",
			want: "",
			want1: &model.AppError{
				Message: "invalid github scope",
				Code:    model.InvalidGithubScope,
				Status:  http.StatusBadRequest},
			args: args{
				existing:   &model.Github{LinkedBy: "user2@example.com"},
				new:        &model.Github{LinkedBy: "user3@example.com"},
				loggedUser: &model.User{Email: "user1@example.com", GHInstallation: &model.GHInstallation{Model: model.Model{ID: "12345"}}},
			},
		},
		{
			name: "invalid account",
			want: "",
			want1: &model.AppError{
				Message: "user- is not linked to github",
				Code:    model.InvalidGithubScope,
				Status:  http.StatusBadRequest},
			args: args{
				existing:   &model.Github{LinkedBy: "user2@example.com"},
				new:        &model.Github{LinkedBy: "user1@example.com"},
				loggedUser: &model.User{Email: "user1@example.com"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := validateGithubScope(tt.args.existing, tt.args.new, tt.args.loggedUser)
			if got != tt.want {
				t.Errorf("validateGithubScope() got = %v, want %v", got, tt.want)
			}

			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("validateGithubScope() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestUpdateLinkProjectGithub(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	u, _ := s.createUser("test-1@example.com", true, "")
	projectID, _ := s.CreateProject(&model.Project{Name: "test"}, u)

	p, _ := s.getProject(projectID, u.ID)
	p.LoadedSettings, _ = model.ParseProjectSettings(p.Settings)
	appErr := s.UpdateProjectGithubLink(u, p)
	if appErr == nil {
		t.Errorf(appErr.Error())
	}

	err := s.createGHInstallation(1234, 5678, "okteto", "user")
	if err != nil {
		t.Fatalf(err.Error())
	}

	err = s.ClaimGHInstallation(u.ID, 1234)
	if err != nil {
		t.Fatalf(err.Error())
	}

	u, err = s.GetUserByGithubID(5678)
	if err != nil {
		t.Fatalf(err.Error())
	}

	appErr = s.UpdateProjectGithubLink(u, p)
	if appErr != nil {
		t.Fatalf(appErr.Error())
	}

	p, _ = s.getProject(projectID, u.ID)
	p.LoadedSettings, _ = model.ParseProjectSettings(p.Settings)
	if p.LoadedSettings.Github.LinkedBy != u.Email {
		t.Errorf("wrong github linked by: %+v", p.LoadedSettings.Github)
	}

	err = s.RemoveProjectGithubLink(p, nil)
	if err != nil {
		t.Fatalf(appErr.Error())
	}

	p, _ = s.getProject(projectID, u.ID)
	p.LoadedSettings, _ = model.ParseProjectSettings(p.Settings)
	if p.LoadedSettings.Github != nil {
		t.Errorf("github scope wasn't removed: %+v", p.LoadedSettings.Github)
	}
}

func BenchmarkGetProjects(b *testing.B) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}
	u, _ := s.createUser("benchmark@example.com", true, "")
	totalProjects := 1000
	for i := 0; i < totalProjects; i++ {
		if _, err := s.CreateProject(&model.Project{Name: fmt.Sprintf("test-%d", i)}, u); err != nil {
			b.Fatal(err.Error())
		}
	}

	b.ResetTimer()
	p, err := s.GetProjects(u)
	if err != nil {
		b.Fatalf(err.Error())
	}

	if len(p) != totalProjects {
		b.Fatalf("expected %d projects but got %d", totalProjects, len(p))
	}
}
