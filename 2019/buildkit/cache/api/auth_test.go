package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bitbucket.org/okteto/okteto/backend/app"
	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/store"
	restful "github.com/emicklei/go-restful"
)

func TestRequestContext(t *testing.T) {
	req := restful.NewRequest(nil)
	u := &model.User{
		Email: "unit-test@example.com",
	}

	p := &model.Project{
		Name: "test-project",
	}

	setAuthenticatedUser(req, u)
	setRequestedProject(req, p)

	ru := getAuthenticatedUser(req)
	if ru != u {
		t.Errorf("failed to retrieve the user from the request")
	}

	rp := getRequestedProject(req)
	if rp != p {
		t.Errorf("failed to retrieve the project from the request")
	}
}

func Test_apiTokenAuthentication(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()

	a := newAPI(&app.Server{DB: db})

	u, appErr := a.app.GetOrCreateUser("user@example.com", false, "")
	if appErr != nil {
		t.Fatalf(appErr.Error())
	}

	projectID, appErr := a.app.CreateProject(&model.Project{Name: fmt.Sprintf("test-%s", u.ID)}, u)
	if appErr != nil {
		t.Fatalf(appErr.Error())
	}

	u2, appErr := a.app.GetOrCreateUser("user2@example.com", false, "")
	if appErr != nil {
		t.Fatalf(appErr.Error())
	}

	projectID2, appErr := a.app.CreateProject(&model.Project{Name: fmt.Sprintf("test-%s", u2.ID)}, u2)
	if appErr != nil {
		t.Fatalf(appErr.Error())
	}

	tests := []struct {
		name     string
		expected int
		token    string
		url      string
	}{
		{
			"all-projects",
			200,
			u.Token,
			"http://localhost/api/v1/projects",
		},
		{
			"single-project",
			200,
			u.Token,
			fmt.Sprintf("http://localhost/api/v1/projects/%s", projectID),
		},
		{
			"non-existing-project",
			403,
			u.Token,
			fmt.Sprintf("http://localhost/api/v1/projects/%s", "non-existing"),
		},
		{
			"not-accesible-project",
			403,
			u.Token,
			fmt.Sprintf("http://localhost/api/v1/projects/%s", projectID2),
		},
		{
			"missing-token",
			401,
			"",
			fmt.Sprintf("http://localhost/api/v1/projects/%s", projectID),
		},
		{
			"bad-token",
			401,
			"blablabla",
			"http://localhost/api/v1/projects",
		},
		{
			"partial-token",
			401,
			u.Token[:len(u.Token)-3],
			fmt.Sprintf("http://localhost/api/v1/projects/%s", projectID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			httpReq, _ := http.NewRequest("GET", tt.url, nil)
			httpReq.Header.Add("Content-Type", "application/json")
			if tt.token != "" {
				httpReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tt.token))
			}

			a.Container.Dispatch(recorder, httpReq)
			if recorder.Result().StatusCode != tt.expected {
				t.Errorf("failed request. expected: %d got %d. \n %s", tt.expected, recorder.Result().StatusCode, recorder.Body.String())
			}
		})
	}
}
