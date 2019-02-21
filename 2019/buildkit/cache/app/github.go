package app

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"bitbucket.org/okteto/okteto/backend/logger"

	"bitbucket.org/okteto/okteto/backend/config"
	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

const (
	defaultManifestPath = "okteto.yaml"
	githubActorID       = "799273b1-b067-4f84-b632-864c543c4dc5"

	installationEvent   = "installation"
	pushEvent           = "push"
	createdInstallation = "created"
	deletedInstallation = "deleted"
)

// GHWebhookPayload is the payload of a github event
type GHWebhookPayload struct {
	Event        string
	Action       string
	Number       int
	Installation *GHInstallation
	Repository   *GHRepo
	Repositories []GHRepo
	Ref          string
	Commit       string `json:"after"`

	// Sender is the account that triggered the action
	Sender *GHAccount
}

// GHInstallation is the payload of a github install event
type GHInstallation struct {
	ID      int
	Account *GHAccount

	// TargetType is the type of account where the application was installed. Can be `Organzation` or `User`
	TargetType string `json:"target_type"`
}

//GHRepo is the information needed to link a github repo to a service
type GHRepo struct {
	ID       int
	Name     string
	FullName string
	Owner    *GHAccount
}

// GHAccount is the payload of a github account and/or organization
type GHAccount struct {
	ID    int
	Login string
}

var githubEvents = make(chan *GHWebhookPayload, 10)

func getManifestFromRepo(installationID int, owner, name, manifest, commit string) (string, error) {
	client, err := newGithubClient(installationID)
	if err != nil {
		return "", err
	}

	content, _, _, err := client.Repositories.GetContents(context.Background(), owner, name, manifest, &github.RepositoryContentGetOptions{Ref: commit})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get contents of %s/%s for ghinstallation-%d", owner, name, installationID)
	}

	contentStr, err := content.GetContent()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get content of %s %s/%s for ghinstallation-%d", manifest, owner, name, installationID)
	}

	return contentStr, nil
}

// newGithubClient returns a github client configured to authenticate as the okteto github app
func newGithubClient(installationID int) (*github.Client, error) {
	ghAppID, ghPrivateKey, err := config.GetGithubApp()
	if err != nil {
		return nil, err
	}

	tr := http.DefaultTransport
	itr, err := ghinstallation.New(tr, ghAppID, installationID, ghPrivateKey)
	if err != nil {
		return nil, err
	}

	// Use installation transport with github.com/google/go-github
	return github.NewClient(&http.Client{Transport: itr}), nil
}

func (s *Server) getGHInstallation(p *model.Project) (*model.GHInstallation, error) {
	if p.GHInstallationID == "" {
		return nil, model.ErrProjectNotLinkedToGithub
	}

	var i model.GHInstallation
	r := s.DB.Where(model.GHInstallation{Model: model.Model{ID: p.GHInstallationID}}).First(&i)
	if r.Error != nil {
		if r.RecordNotFound() {
			return nil, errors.Wrapf(model.ErrNotFound, "ghinstallation-%s not found in the DB", p.GHInstallationID)
		}

		return nil, errors.Wrapf(r.Error, "failed to get ghinstallation for project-%s", p.ID)
	}

	return &i, nil
}

// createGHInstallation saves the integration the DB
func (s *Server) createGHInstallation(installationID int, githubID int, githubLogin string, targetType model.GHScope) error {
	if installationID == 0 {
		return errors.New("installationID is not set")
	}

	if githubID == 0 {
		return errors.New("githubUserID is not set")
	}

	installation := model.GHInstallation{
		InstallationID: installationID,
		GithubID:       githubID,
		GithubLogin:    githubLogin,
		Scope:          targetType,
	}

	err := s.DB.Create(&installation).Error
	if err != nil {
		return err
	}

	return nil
}

// deleteGHInstallation removes the installation from okteto, and unlinks all the existing projects and services
// This event starts from Github
func (s *Server) deleteGHInstallation(installationID int) error {
	if installationID == 0 {
		return errors.New("installationID is not set")
	}

	tx := s.DB.Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "couldn't start transaction")
	}

	i := model.GHInstallation{InstallationID: installationID}
	r := tx.Where(i).First(&i)

	if r.Error != nil {
		tx.Rollback()
		if r.RecordNotFound() {
			return errors.Wrapf(model.ErrNotFound, "ghinstallation-%d not found in the DB", installationID)
		}

		return errors.Wrapf(r.Error, "failed to get ghinstallation-%d", installationID)
	}

	var projects []model.Project
	err := tx.Where(model.Project{GHInstallationID: i.ID}).Find(&projects).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrapf(err, "failed to get projects with ghinstallation-%d", installationID)
	}

	for _, p := range projects {
		err := s.RemoveProjectGithubLink(&p, tx)
		if err != nil {
			tx.Rollback()
			return errors.Wrapf(err, "Failed to remove github links from project-%s", p.ID)
		}
	}

	r = tx.Delete(&i)
	if r.Error != nil {
		tx.Rollback()
		if r.RecordNotFound() {
			return errors.Wrapf(model.ErrNotFound, "ghinstallation-%d not found in the DB", installationID)
		}

		return errors.Wrapf(r.Error, "failed to delete ghinstallation-%d", installationID)
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "couldn't commit transaction for RemoveProjectGithubLink")
	}

	return nil
}

// ClaimGHInstallation links the GH installation with an okteto user
func (s *Server) ClaimGHInstallation(userID string, installationID int) error {
	if installationID == 0 {
		return errors.New("installationID is not set")
	}

	i := model.GHInstallation{InstallationID: installationID}
	r := s.DB.Model(&i).Where(i).Updates(model.GHInstallation{UserID: userID})

	if r.Error != nil {
		if r.RecordNotFound() {
			return errors.Wrapf(model.ErrNotFound, "ghinstallation-%d not found in the DB", installationID)
		}

		return errors.Wrapf(r.Error, "failed to get ghinstallation-%d", installationID)
	}

	if r.RowsAffected == 0 {
		return errors.Wrapf(model.ErrNotFound, "ghinstallation-%d not found in the DB", installationID)
	}

	return nil
}

// syncManifestFromGH updates the manifest of all the services that are linked to this repo/branch combination
func (s *Server) syncManifestFromGH(installationID, repositoryID int, repositoryOwner, repositoryName, branch, commit, author string) error {
	link := model.GHRepoLink{InstallationID: installationID, RepositoryID: repositoryID, Branch: branch}
	r := s.DB.Where(link).First(&link)
	if r.Error != nil {
		if r.RecordNotFound() {
			return errors.Wrapf(model.ErrNotFound, "ghinstallation-%d repository-%d branch-%s not found in the DB", installationID, repositoryID, branch)
		}

		return errors.Wrap(r.Error, "failed to query for GHRepoLink")
	}

	var services []model.Service
	r = s.DB.Where(model.Service{GHRepoLinkID: link.ID}).Find(&services)
	if r.Error != nil {
		if r.RecordNotFound() {
			return nil
		}

		return errors.Wrap(r.Error, "failed to query for github linked services")
	}

	if len(services) > 0 {
		content, err := getManifestFromRepo(installationID, repositoryOwner, repositoryName, link.Manifest, commit)
		if err != nil {
			return err
		}

		encodedManifest := base64.StdEncoding.EncodeToString([]byte(content))
		logMessage := fmt.Sprintf("Updated manifest due to commit #%s by %s", commit, author)
		for _, svc := range services {
			err := s.UpdateManifest(svc.ProjectID, svc.ID, encodedManifest, githubActorID, logMessage)
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to update manifest of service-%s", svc.ID))
			}
		}
	}

	return nil
}

// LinkGHRepositoryToService links a gh repository to a existing okteto service
func (s *Server) LinkGHRepositoryToService(p *model.Project, serviceID string, owner, name, branch, manifest string) error {

	i, err := s.getGHInstallation(p)
	if err != nil {
		return err
	}

	client, err := newGithubClient(i.InstallationID)
	if err != nil {
		return err
	}

	repo, _, err := client.Repositories.Get(context.Background(), owner, name)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s/%s via the api", owner, name)
	}

	svc, appErr := s.getService(p, serviceID)
	if err != nil {
		return errors.Wrapf(appErr, "failed to get service-%s", serviceID)
	}

	if manifest == "" {
		manifest = defaultManifestPath
	}

	if branch == "" {
		branch = repo.GetDefaultBranch()
	}

	// get canonical form of the branch name
	branch = getCanonicalBranchName(branch)

	repoLink := model.GHRepoLink{
		InstallationID: i.InstallationID,
		RepositoryID:   int(repo.GetID()),
		Branch:         branch,
		Manifest:       manifest,
	}

	tx := s.DB.Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to start transaction")
	}

	err = tx.Where(repoLink).FirstOrCreate(&repoLink).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrapf(err, "failed to create or find repo link for service-%s", serviceID)
	}

	err = tx.Model(&svc).Updates(&model.Service{GHRepoLinkID: repoLink.ID}).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrapf(err, "failed to update github link ID for service-%s", serviceID)
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return errors.Wrapf(err, "failed to commit transaction when linking service-%s", serviceID)
	}

	return nil
}

//UnlinkGHRepositoryToService removes the github link to a service
func (s *Server) UnlinkGHRepositoryToService(p *model.Project, serviceID string) error {
	svc, appErr := s.getService(p, serviceID)
	if appErr != nil {
		if appErr.Code == model.EntityNotFound {
			return errors.Wrapf(model.ErrNotFound, "service-%s not in the DB", serviceID)
		}
	}

	svc.GHRepoLinkID = ""
	err := s.DB.Save(&svc).Error
	if err != nil {
		return errors.Wrapf(err, "failed to unlink service-%s", serviceID)
	}

	return nil
}

// GetGHRepositories returns all repositories p can see
func (s *Server) GetGHRepositories(p *model.Project) ([]*github.Repository, error) {
	gi, err := s.getGHInstallation(p)
	if err != nil {
		return nil, err
	}

	client, err := newGithubClient(gi.InstallationID)
	if err != nil {
		return nil, err
	}

	var repositories []*github.Repository
	if gi.Scope == model.GHUser {
		repositories, _, err = client.Repositories.List(context.Background(), gi.GithubLogin, &github.RepositoryListOptions{})
	} else {
		repositories, _, err = client.Repositories.ListByOrg(context.Background(), gi.GithubLogin, &github.RepositoryListByOrgOptions{})
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get repositories for project-%s", p.ID)
	}
	return repositories, nil
}

func (s *Server) handleInstallation(payload *GHWebhookPayload) {
	if payload.Action == createdInstallation {
		err := s.createGHInstallation(payload.Installation.ID, payload.Sender.ID, payload.Installation.Account.Login, model.GHScope(payload.Installation.TargetType))
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to create ghinstallation"))
			return
		}
	} else if payload.Action == deletedInstallation {
		err := s.deleteGHInstallation(payload.Installation.ID)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to delete ghinstallation"))
			return
		}
	}
}

func (s *Server) handlePush(webhook *GHWebhookPayload) {
	if webhook.Ref == "" {
		logger.Error(errors.New("webhook didn't have a ref"))
		return
	}

	if webhook.Commit == "" {
		logger.Error(errors.New("webhook didn't have a commit"))
		return
	}

	err := s.syncManifestFromGH(webhook.Installation.ID, webhook.Repository.ID, webhook.Repository.Owner.Login, webhook.Repository.Name, webhook.Ref, webhook.Commit, webhook.Sender.Login)
	if err != nil {
		if err == model.ErrSHAMismatch {
			return
		}

		if err == model.ErrRefMismatch {
			return
		}

		logger.Error(errors.Wrap(err, "sync from gh failed"))
	}
}

func (s *Server) processGithubEvents() {
	for {
		select {
		case ev := <-githubEvents:
			switch ev.Event {
			case installationEvent:
				s.handleInstallation(ev)
			case pushEvent:
				s.handlePush(ev)
			default:
				logger.Error(fmt.Errorf("unknown github event queued: %s", ev.Event))
			}
		}
	}
}

// QueueGithubEvent adds a github event to be processed
func (s *Server) QueueGithubEvent(w *GHWebhookPayload) {
	githubEvents <- w
}

//IsGithubEventSupported returns true if the event is supported
func IsGithubEventSupported(event string) bool {
	if event == installationEvent || event == pushEvent {
		return true
	}

	return false
}

func getCanonicalBranchName(branch string) string {
	branch = strings.TrimPrefix(branch, "refs/heads/")
	return fmt.Sprintf("refs/heads/%s", branch)
}
