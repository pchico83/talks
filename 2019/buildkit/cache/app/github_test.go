package app

import (
	"encoding/json"
	"testing"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/store"
)

func TestGHInstall(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}

	payloadBytes := []byte(`{"action":"created","installation":{"id":277287,"account":{"login":"rberrelleza","id":475313,"node_id":"MDQ6VXNlcjQ3NTMxMw==","avatar_url":"https://avatars0.githubusercontent.com/u/475313?v=4","gravatar_id":"","url":"https://api.github.com/users/rberrelleza","html_url":"https://github.com/rberrelleza","followers_url":"https://api.github.com/users/rberrelleza/followers","following_url":"https://api.github.com/users/rberrelleza/following{/other_user}","gists_url":"https://api.github.com/users/rberrelleza/gists{/gist_id}","starred_url":"https://api.github.com/users/rberrelleza/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/rberrelleza/subscriptions","organizations_url":"https://api.github.com/users/rberrelleza/orgs","repos_url":"https://api.github.com/users/rberrelleza/repos","events_url":"https://api.github.com/users/rberrelleza/events{/privacy}","received_events_url":"https://api.github.com/users/rberrelleza/received_events","type":"User","site_admin":false},"repository_selection":"selected","access_tokens_url":"https://api.github.com/installations/277287/access_tokens","repositories_url":"https://api.github.com/installation/repositories","html_url":"https://github.com/settings/installations/277287","app_id":15718,"target_id":475313,"target_type":"User","permissions":{"statuses":"write","deployments":"write","single_file":"write","contents":"read","pull_requests":"read","metadata":"read"},"events":["commit_comment","deployment","deployment_status","pull_request","status"],"created_at":1533653612,"updated_at":1533653612,"single_file_name":"okteto.yaml"},"repositories":[{"id":143887075,"name":"app-test","full_name":"rberrelleza/app-test","private":false}],"sender":{"login":"rberrelleza","id":475313,"node_id":"MDQ6VXNlcjQ3NTMxMw==","avatar_url":"https://avatars0.githubusercontent.com/u/475313?v=4","gravatar_id":"","url":"https://api.github.com/users/rberrelleza","html_url":"https://github.com/rberrelleza","followers_url":"https://api.github.com/users/rberrelleza/followers","following_url":"https://api.github.com/users/rberrelleza/following{/other_user}","gists_url":"https://api.github.com/users/rberrelleza/gists{/gist_id}","starred_url":"https://api.github.com/users/rberrelleza/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/rberrelleza/subscriptions","organizations_url":"https://api.github.com/users/rberrelleza/orgs","repos_url":"https://api.github.com/users/rberrelleza/repos","events_url":"https://api.github.com/users/rberrelleza/events{/privacy}","received_events_url":"https://api.github.com/users/rberrelleza/received_events","type":"User","site_admin":false}}`)

	payload := &GHWebhookPayload{}
	err := json.Unmarshal(payloadBytes, payload)
	if err != nil {
		t.Fatal(err.Error())
	}

	if payload.Action != createdInstallation {
		t.Errorf("action wasn't parsed: %+v", payload)
	}

	if payload.Installation.Account.Login != "rberrelleza" {
		t.Errorf("Installation.Account.Login wasn't parsed: %+v", payload)
	}

	if payload.Repositories[0].Name != "app-test" {
		t.Errorf("Repositories[0].Name wasn't parsed: %+v", payload)
	}

	err = s.createGHInstallation(payload.Installation.ID, payload.Installation.Account.ID, payload.Installation.Account.Login, model.GHScope(payload.Installation.TargetType))
	if err != nil {
		t.Fatalf(err.Error())
	}

	var i model.GHInstallation
	err = s.DB.Where(model.GHInstallation{InstallationID: payload.Installation.ID}).Find(&i).Error
	if err != nil {
		t.Fatalf(err.Error())
	}

	gi, err := s.getGHInstallation(&model.Project{GHInstallationID: i.ID})
	if err != nil {
		t.Fatalf(err.Error())
	}

	if gi.ID != i.ID {
		t.Fatalf("expecting: %+v, got %+v", i, gi)
	}

}

func TestPushEventParsing(t *testing.T) {
	eventBody := []byte(`{"ref":"refs/heads/master","before":"e752c8ddda96a4af007774d29536f6b17e7e1ba7","after":"c1cc3fdbbf61f9e23fbc3456c761284ca7db16b3","created":false,"deleted":false,"forced":false,"base_ref":null,"compare":"https://github.com/rberrelleza/app-test/compare/e752c8ddda96...c1cc3fdbbf61","commits":[{"id":"c1cc3fdbbf61f9e23fbc3456c761284ca7db16b3","tree_id":"a0c232f2c6c8deba7767098df05566b3b4e07ce6","distinct":true,"message":"Update README.md","timestamp":"2018-08-13T23:25:29+02:00","url":"https://github.com/rberrelleza/app-test/commit/c1cc3fdbbf61f9e23fbc3456c761284ca7db16b3","author":{"name":"Ramiro Berrelleza","email":"rberrelleza@gmail.com","username":"rberrelleza"},"committer":{"name":"GitHub","email":"noreply@github.com","username":"web-flow"},"added":[],"removed":[],"modified":["README.md"]}],"head_commit":{"id":"c1cc3fdbbf61f9e23fbc3456c761284ca7db16b3","tree_id":"a0c232f2c6c8deba7767098df05566b3b4e07ce6","distinct":true,"message":"Update README.md","timestamp":"2018-08-13T23:25:29+02:00","url":"https://github.com/rberrelleza/app-test/commit/c1cc3fdbbf61f9e23fbc3456c761284ca7db16b3","author":{"name":"Ramiro Berrelleza","email":"rberrelleza@gmail.com","username":"rberrelleza"},"committer":{"name":"GitHub","email":"noreply@github.com","username":"web-flow"},"added":[],"removed":[],"modified":["README.md"]},"repository":{"id":143887075,"node_id":"MDEwOlJlcG9zaXRvcnkxNDM4ODcwNzU=","name":"app-test","full_name":"rberrelleza/app-test","owner":{"name":"rberrelleza","email":"rberrelleza@gmail.com","login":"rberrelleza","id":475313,"node_id":"MDQ6VXNlcjQ3NTMxMw==","avatar_url":"https://avatars0.githubusercontent.com/u/475313?v=4","gravatar_id":"","url":"https://api.github.com/users/rberrelleza","html_url":"https://github.com/rberrelleza","followers_url":"https://api.github.com/users/rberrelleza/followers","following_url":"https://api.github.com/users/rberrelleza/following{/other_user}","gists_url":"https://api.github.com/users/rberrelleza/gists{/gist_id}","starred_url":"https://api.github.com/users/rberrelleza/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/rberrelleza/subscriptions","organizations_url":"https://api.github.com/users/rberrelleza/orgs","repos_url":"https://api.github.com/users/rberrelleza/repos","events_url":"https://api.github.com/users/rberrelleza/events{/privacy}","received_events_url":"https://api.github.com/users/rberrelleza/received_events","type":"User","site_admin":false},"private":false,"html_url":"https://github.com/rberrelleza/app-test","description":null,"fork":false,"url":"https://github.com/rberrelleza/app-test","forks_url":"https://api.github.com/repos/rberrelleza/app-test/forks","keys_url":"https://api.github.com/repos/rberrelleza/app-test/keys{/key_id}","collaborators_url":"https://api.github.com/repos/rberrelleza/app-test/collaborators{/collaborator}","teams_url":"https://api.github.com/repos/rberrelleza/app-test/teams","hooks_url":"https://api.github.com/repos/rberrelleza/app-test/hooks","issue_events_url":"https://api.github.com/repos/rberrelleza/app-test/issues/events{/number}","events_url":"https://api.github.com/repos/rberrelleza/app-test/events","assignees_url":"https://api.github.com/repos/rberrelleza/app-test/assignees{/user}","branches_url":"https://api.github.com/repos/rberrelleza/app-test/branches{/branch}","tags_url":"https://api.github.com/repos/rberrelleza/app-test/tags","blobs_url":"https://api.github.com/repos/rberrelleza/app-test/git/blobs{/sha}","git_tags_url":"https://api.github.com/repos/rberrelleza/app-test/git/tags{/sha}","git_refs_url":"https://api.github.com/repos/rberrelleza/app-test/git/refs{/sha}","trees_url":"https://api.github.com/repos/rberrelleza/app-test/git/trees{/sha}","statuses_url":"https://api.github.com/repos/rberrelleza/app-test/statuses/{sha}","languages_url":"https://api.github.com/repos/rberrelleza/app-test/languages","stargazers_url":"https://api.github.com/repos/rberrelleza/app-test/stargazers","contributors_url":"https://api.github.com/repos/rberrelleza/app-test/contributors","subscribers_url":"https://api.github.com/repos/rberrelleza/app-test/subscribers","subscription_url":"https://api.github.com/repos/rberrelleza/app-test/subscription","commits_url":"https://api.github.com/repos/rberrelleza/app-test/commits{/sha}","git_commits_url":"https://api.github.com/repos/rberrelleza/app-test/git/commits{/sha}","comments_url":"https://api.github.com/repos/rberrelleza/app-test/comments{/number}","issue_comment_url":"https://api.github.com/repos/rberrelleza/app-test/issues/comments{/number}","contents_url":"https://api.github.com/repos/rberrelleza/app-test/contents/{+path}","compare_url":"https://api.github.com/repos/rberrelleza/app-test/compare/{base}...{head}","merges_url":"https://api.github.com/repos/rberrelleza/app-test/merges","archive_url":"https://api.github.com/repos/rberrelleza/app-test/{archive_format}{/ref}","downloads_url":"https://api.github.com/repos/rberrelleza/app-test/downloads","issues_url":"https://api.github.com/repos/rberrelleza/app-test/issues{/number}","pulls_url":"https://api.github.com/repos/rberrelleza/app-test/pulls{/number}","milestones_url":"https://api.github.com/repos/rberrelleza/app-test/milestones{/number}","notifications_url":"https://api.github.com/repos/rberrelleza/app-test/notifications{?since,all,participating}","labels_url":"https://api.github.com/repos/rberrelleza/app-test/labels{/name}","releases_url":"https://api.github.com/repos/rberrelleza/app-test/releases{/id}","deployments_url":"https://api.github.com/repos/rberrelleza/app-test/deployments","created_at":1533652987,"updated_at":"2018-08-13T21:18:49Z","pushed_at":1534195529,"git_url":"git://github.com/rberrelleza/app-test.git","ssh_url":"git@github.com:rberrelleza/app-test.git","clone_url":"https://github.com/rberrelleza/app-test.git","svn_url":"https://github.com/rberrelleza/app-test","homepage":null,"size":2,"stargazers_count":0,"watchers_count":0,"language":"Python","has_issues":true,"has_projects":true,"has_downloads":true,"has_wiki":true,"has_pages":false,"forks_count":0,"mirror_url":null,"archived":false,"open_issues_count":1,"license":null,"forks":0,"open_issues":1,"watchers":0,"default_branch":"master","stargazers":0,"master_branch":"master"},"pusher":{"name":"rberrelleza","email":"rberrelleza@gmail.com"},"sender":{"login":"rberrelleza","id":475313,"node_id":"MDQ6VXNlcjQ3NTMxMw==","avatar_url":"https://avatars0.githubusercontent.com/u/475313?v=4","gravatar_id":"","url":"https://api.github.com/users/rberrelleza","html_url":"https://github.com/rberrelleza","followers_url":"https://api.github.com/users/rberrelleza/followers","following_url":"https://api.github.com/users/rberrelleza/following{/other_user}","gists_url":"https://api.github.com/users/rberrelleza/gists{/gist_id}","starred_url":"https://api.github.com/users/rberrelleza/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/rberrelleza/subscriptions","organizations_url":"https://api.github.com/users/rberrelleza/orgs","repos_url":"https://api.github.com/users/rberrelleza/repos","events_url":"https://api.github.com/users/rberrelleza/events{/privacy}","received_events_url":"https://api.github.com/users/rberrelleza/received_events","type":"User","site_admin":false},"installation":{"id":284405}}`)
	event := &GHWebhookPayload{}
	err := json.Unmarshal(eventBody, event)
	if err != nil {
		t.Fatal(err.Error())
	}

	if event.Ref == "" {
		t.Errorf("ref is missing")
	}

	if event.Commit == "" {
		t.Errorf("head is missing")
	}

	if event.Repository.ID == 0 {
		t.Errorf("reposotiry id is missing")
	}
}

func TestUnlinkGHRepositoryToService(t *testing.T) {
	db := store.NewMemoryStore()
	defer db.Close()
	s := Server{DB: db}

	u, _ := s.createUser("user@example.com", true, "")
	pID, _ := s.CreateProject(&model.Project{Name: "test"}, u)
	p, _ := s.GetProject(pID, u.ID)
	svc, _ := s.CreateService(p, &model.Service{Manifest: httpService, GHRepoLinkID: "12345"}, u)

	svc, appErr := s.GetServiceByID(svc.ID)
	if appErr != nil {
		t.Fatalf(appErr.Error())
	}

	if svc.GHRepoLinkID == "" {
		t.Fatalf("service wasn't linked")
	}

	err := s.UnlinkGHRepositoryToService(p, svc.ID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	svc, appErr = s.GetServiceByID(svc.ID)
	if appErr != nil {
		t.Fatalf(appErr.Error())
	}

	if svc.GHRepoLinkID != "" {
		t.Fatalf("service wasn't unlinked: '%s'", svc.GHRepoLinkID)
	}
}

func Test_getCanonicalBranchName(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   string
	}{
		{
			name:   "just name",
			branch: "master",
			want:   "refs/heads/master",
		},
		{
			name:   "full ref",
			branch: "refs/heads/master",
			want:   "refs/heads/master",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCanonicalBranchName(tt.branch); got != tt.want {
				t.Errorf("getCanonicalBranchName() = %v, want %v", got, tt.want)
			}
		})
	}
}
