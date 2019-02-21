package app

import (
	"reflect"
	"testing"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/store"
)

func TestCreateUser(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	s := Server{DB: db, Email: NewMail("hello@okteto.com", &NoopMail{})}

	u, appErr := s.GetOrCreateUser("user@example.com", false, "")
	if appErr != nil {
		t.Fatalf("%+v", appErr)
	}

	if u.ID == "" {
		t.Errorf("ID wasn't set")
	}

	if u.Email != "user@example.com" {
		t.Errorf("email wasn't set")
	}

	if u.Verified == true {
		t.Errorf("user was wrongfully verified")
	}

	if u.GHInstallation != nil {
		t.Errorf("GHInstallation was set")
	}

	after, appErr := s.GetOrCreateUser(u.Email, false, "")
	if appErr != nil {
		t.Fatalf("%+v", appErr)
	}

	if !reflect.DeepEqual(*u, *after) {
		t.Errorf("user was different on the second get: \n%+v \n%+v", *u, *after)
	}

}

func TestInviteUserAlreadyExists(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	e := &testEmailSender{}
	s := Server{Email: NewMail("hello@okteto.com", e), DB: db}

	authUser, _ := s.createUser("first@example.com", true, "")
	u, _ := s.createUser("test-1@example.com", true, "")

	p := &model.Project{Name: "project-1"}
	projectID, appErr := s.CreateProject(p, authUser)
	if appErr != nil {
		t.Fatalf("failed to create project: %+v", appErr)
	}

	appErr = s.InviteUser(authUser, u.Email, projectID, model.ProjectRoleUser)
	if appErr == nil {
		t.Fatalf("shouldn't re-invite existing users")
	}

	if appErr.Code != model.InvalidEmail {
		t.Errorf("got the wrong error code")
	}

	if e.sent != 0 {
		t.Errorf("EmailProvider was called")
	}
}

func TestInviteUserAlreadyExistsButUnverified(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	e := &testEmailSender{}
	s := Server{Email: NewMail("hello@okteto.com", e), DB: db}

	authUser, _ := s.createUser("first@example.com", true, "")
	u, _ := s.createUser("test-1@example.com", false, "")

	p := &model.Project{Name: "project-1"}
	projectID, appErr := s.CreateProject(p, authUser)
	if appErr != nil {
		t.Fatalf("failed to create project: %+v", appErr)
	}

	appErr = s.InviteUser(authUser, u.Email, projectID, model.ProjectRoleUser)
	if appErr != nil {
		t.Errorf("undexpected error: %s", appErr.Error())
	}

	if e.sent != 1 {
		t.Errorf("email provider was not called")
	}

	if e.email != u.Email {
		t.Errorf("email provider didn't receive the corrrect email. Got: %s, expecting: %s", e.email, u.Email)
	}

}

func TestInviteNewUser(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	e := &testEmailSender{}
	s := Server{Email: NewMail("hello@okteto.com", e), DB: db}

	u, _ := s.createUser("test-1@example.com", true, "")

	projectID, appErr := s.CreateProject(&model.Project{Name: "project1"}, u)

	if appErr != nil {
		t.Fatalf("%+v", appErr)
	}

	appErr = s.InviteUser(u, "test-2@example.com", projectID, model.ProjectRoleUser)
	if appErr != nil {
		t.Fatalf("user wasn't invited: %+v", appErr)
	}

	if e.sent != 1 {
		t.Errorf("email provider was called %d times", e.sent)
	}

	if e.email != "test-2@example.com" {
		t.Errorf("email provider didn't receive the corrrect email")
	}

	invited, appErr := s.GetUser("test-2@example.com")

	if invited == nil {
		t.Errorf("user wasn't added to the store: %+v", appErr)
	}

	if invited.Verified == true {
		t.Errorf("invited user was not marked as unverified: %+v", invited)
	}
}

func TestDeleteUser(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	s := Server{Email: NewMail("hello@okteto.com", &NoopMail{}), DB: db}

	user := model.NewUser("first@example.com")
	err := s.DB.Create(&user).Error
	if err != nil {
		t.Fatalf("failed to create user: %s", err)
	}

	user, err = s.GetUserByToken(user.Token)
	if err != nil {
		t.Fatalf("failed to get user by token: %s", err)
	}

	appErr := s.DeleteUser(user)
	if appErr != nil {
		t.Fatalf("failed to delete user: %+v", appErr)
	}

	u, err := s.GetUserByToken(user.Token)
	if err == nil {
		t.Fatalf("user wasn't deleted, got: %+v", u)
	}
}

func TestGetUserByGithubID(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{Email: NewMail("hello@okteto.com", &NoopMail{}), DB: db}
	_, err := s.GetUserByGithubID(1)

	if err == nil {
		t.Fatalf("didn't receive an error")
	}

	u, appErr := s.createUser("user@example.com", true, "")
	if appErr != nil {
		t.Fatalf("received an error when creating: %s", appErr.Error())
	}

	installationID := 12345
	githubID := 145678

	err = s.createGHInstallation(installationID, githubID, "okteto", "user")
	if err != nil {
		t.Fatalf("failed to create gh installation: %s", err.Error())
	}

	err = s.ClaimGHInstallation(u.ID, installationID)
	if err != nil {
		t.Fatalf("failed to link gh installation: %s", err.Error())
	}

	i := model.GHInstallation{InstallationID: installationID}
	err = s.DB.Where(i).Find(&i).Error
	if err != nil {
		t.Fatalf("received an error when getting ghinstallation: %s", err.Error())
	}

	if i.ID == "" {
		t.Fatalf("ghinstallation doesn't have an ID: %+v", i)
	}

	if i.UserID != u.ID {
		t.Fatalf("ghinstallation wasn't linked to a user: %+v", i)
	}

	g, err := s.GetUserByGithubID(githubID)
	if err != nil {
		t.Fatalf("received an error when getting with githubID: %s", err.Error())
	}

	if g.ID != u.ID {
		t.Fatalf("received the wrong user. Got %+v, Expected %+v", g, u)
	}

	if g.GHInstallation == nil {
		t.Fatal("ghinstallation was nil")
	}

	if g.GHInstallation.ID != i.ID {
		t.Fatalf("received the wrong ghinstallation. Got %s, Expected %s", g.GHInstallation.ID, i.ID)
	}
}

func TestLinkToMixpanel(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	s := Server{DB: db, Email: NewMail("hello@okteto.com", &NoopMail{})}

	u, appErr := s.GetOrCreateUser("user@example.com", false, "")
	if appErr != nil {
		t.Fatalf("%+v", appErr)
	}

	mixpanel := "1234567899"
	err := s.LinkToMixpanel(u, mixpanel)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := s.GetUserByID(u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if updated.Mixpanel != mixpanel {
		t.Fatalf("Expected: %s, got: %s", mixpanel, updated.Mixpanel)
	}
}

type testEmailSender struct {
	expectedError error
	email         string
	sent          int
}

func (t *testEmailSender) send(from, title, body, bodyHTML string, to ...string) error {
	t.sent++
	t.email = to[0]

	return t.expectedError
}
