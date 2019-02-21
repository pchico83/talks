package app

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/providers/k8/user"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"bitbucket.org/okteto/okteto/backend/config"
	"bitbucket.org/okteto/okteto/backend/model"
)

//Credentials represents kubernetes dev credentials for a project
type Credentials struct {
	User      string `json:"user,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Password  string `json:"password,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	CaCert    string `json:"ca_cert,omitempty"`
}

//Credentials returns the kubernetes credentials a given project
func (s *Server) Credentials(project *model.Project) (*Credentials, *model.AppError) {
	e := s.buildEnvironment(project)

	e.Provider.LoadDefaultCluster()
	devName := user.DevName(e)
	token, err := user.GetServiceAccountCredential(e)
	if err != nil {
		return nil, &model.AppError{Status: 500, Code: model.InternalServerError, Message: err.Error()}
	}
	return &Credentials{
		User:      devName,
		Password:  token,
		Namespace: e.Name,
		Endpoint:  project.LoadedSettings.Provider.Endpoint,
		CaCert:    project.LoadedSettings.Provider.CaCert,
	}, nil
}

func getDefaultSettings(userEmail string) *model.ProjectSettings {
	settings := model.ProjectSettings{
		Administrators: []string{userEmail},
		Provider:       &model.Provider{Type: model.Demo},
	}

	return &settings
}

//CreateProject inserts a project in the DB
func (s *Server) CreateProject(project *model.Project, user *model.User) (string, *model.AppError) {
	appErr := project.Validate()
	if appErr != nil {
		return "", appErr
	}

	project.DNSName = strings.ToLower(project.Name)
	settings := getDefaultSettings(user.Email)
	project.LoadedSettings = settings
	project.Settings = settings.Base64Encode()

	var i = 0
	var tx *gorm.DB
	var err error

	for {
		tx = s.DB.Begin()
		if tx.Error != nil {
			return "", &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: tx.Error.Error()}
		}

		err = tx.Create(&project).Error
		if err != nil {
			tx.Rollback()
			if checkForUniqueDNSError(err) {
				i++
				project.DNSName = fmt.Sprintf("%s-%d", project.Name, i)
				continue
			}

			break
		}

		err = tx.Create(&model.ProjectACL{ProjectID: project.ID, Role: model.ProjectRoleAdmin, UserID: user.ID}).Error
		if err != nil {
			break
		}

		err = tx.Commit().Error
		if err != nil {
			break
		}

		invitedUsers := findInvitedUsers([]string{user.Email}, append(project.LoadedSettings.Administrators, project.LoadedSettings.Users...))
		s.sendProjectInvite(invitedUsers, project.Name)
		go s.newProjectNotification(user)
		return project.ID, nil

	}

	if tx != nil {
		tx.Rollback()
	}

	return "", &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: err.Error()}
}

func (s *Server) getProject(projectID string, userID string) (*model.Project, *model.AppError) {
	rows, err := s.DB.Raw(
		"SELECT p.id, p.name, p.settings, p.dns_name, a.role, p.gh_installation_id FROM projects as p INNER JOIN project_acls as a ON p.id = a.project_id WHERE a.user_id = ? AND a.project_id = ? AND p.deleted_at IS NULL",
		userID, projectID).Rows()

	if err != nil {
		return nil, &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: err.Error()}
	}

	defer rows.Close()
	var project model.Project

	for rows.Next() {
		err = rows.Scan(&project.ID, &project.Name, &project.Settings, &project.DNSName, &project.Role, &project.GHInstallationID)
		if err != nil {
			return nil, &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: err.Error()}
		}

		break
	}

	if project.ID == "" {
		return nil, &model.AppError{Status: http.StatusNotFound, Code: model.EntityNotFound}
	}

	return &project, nil
}

func (s *Server) getProjectACLS(projectID string) ([]model.ProjectACL, error) {
	var projectACLs []model.ProjectACL
	rows, err := s.DB.Raw("SELECT p.project_id, u.email, p.role FROM project_acls as p INNER JOIN users as u ON p.user_id = u.id WHERE project_id = ?", projectID).Rows()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var acl model.ProjectACL
		err := rows.Scan(&acl.ProjectID, &acl.UserEmail, &acl.Role)
		if err != nil {
			return nil, err
		}

		projectACLs = append(projectACLs, acl)
	}

	if len(projectACLs) == 0 {
		logger.Error(fmt.Errorf("project-%s doesn't have ACLS", projectID))
	}

	return projectACLs, nil
}

// GetProject returns the project that matches the ID, or an error
func (s *Server) GetProject(projectID string, userID string) (*model.Project, *model.AppError) {
	if projectID == "" {
		return nil, &model.AppError{Status: http.StatusBadRequest, Code: model.MissingID}
	}

	project, appErr := s.getProject(projectID, userID)

	if appErr != nil {
		return nil, appErr
	}

	project.LoadedSettings, appErr = model.ParseProjectSettings(project.Settings)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to decode project-%s settings, this is most likely a bug or a project schema change issue", project.ID))
		return nil, appErr
	}

	if project.Role == model.ProjectRoleAdmin {
		// This block replaces the users in the manifest with the contents of the ACL. Once
		// we move away from YAML for settings this won't be needed
		projectACLs, err := s.getProjectACLS(projectID)
		if err != nil {
			return nil, &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: err.Error()}
		}

		project.LoadedSettings.Users = make([]string, 0)
		project.LoadedSettings.Administrators = make([]string, 0)
		for i := range projectACLs {
			if projectACLs[i].Role == model.ProjectRoleAdmin {
				project.LoadedSettings.Administrators = append(project.LoadedSettings.Administrators, projectACLs[i].UserEmail)
			} else {
				project.LoadedSettings.Users = append(project.LoadedSettings.Users, projectACLs[i].UserEmail)
			}
		}

		project.Settings = project.LoadedSettings.Base64Encode()
	}

	return project, nil
}

// GetProjects returns all the projects of the authenticated user
func (s *Server) GetProjects(u *model.User) ([]model.Project, *model.AppError) {
	projects := make([]model.Project, 0)

	result := s.DB.Raw(
		"SELECT p.id, p.name, p.settings, p.dns_name, p.gh_installation_id, a.role FROM projects as p INNER JOIN project_acls as a ON p.id = a.project_id WHERE a.user_id = ? AND p.deleted_at IS NULL",
		u.ID)

	rows, err := result.Rows()
	if err != nil {
		if result.RecordNotFound() {
			return projects, nil
		}

		return nil, &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: err.Error()}
	}

	defer rows.Close()

	for rows.Next() {
		var project model.Project
		err = rows.Scan(&project.ID, &project.Name, &project.Settings, &project.DNSName, &project.GHInstallationID, &project.Role)
		if err != nil {
			return nil, &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: err.Error()}
		}

		settings, appErr := model.ParseProjectSettings(project.Settings)
		if appErr != nil {
			return nil, appErr
		}

		project.LoadedSettings = settings
		projects = append(projects, project)
	}

	return projects, nil
}

// UpdateProjectSettings updates the project or throws an error. This only
// updates settings and ACLS
func (s *Server) UpdateProjectSettings(loggedUser *model.User, existing *model.Project, newSettings string) ([]model.ProjectACL, *model.AppError) {
	if newSettings == "" {
		return nil, &model.AppError{Code: model.MissingProjectSettings, Status: http.StatusBadRequest, Message: "settings is empty"}
	}

	previousUsers := existing.LoadedSettings.Administrators
	if previousUsers == nil {
		previousUsers = []string{}
	}

	if existing.LoadedSettings.Users != nil {
		previousUsers = append(previousUsers, existing.LoadedSettings.Users...)
	}

	settings, appErr := model.ParseProjectSettings(newSettings)
	if appErr != nil {
		return nil, appErr
	}
	ghInstallationID, appErr := validateGithubScope(existing.LoadedSettings.Github, settings.Github, loggedUser)
	if appErr != nil {
		return nil, appErr
	}

	existing.GHInstallationID = ghInstallationID

	acls := settings.ToProjectACLs(existing.ID)
	existingUsers, pendingUsers, appErr := s.validateACLs(acls)
	if appErr != nil {
		return nil, appErr
	}

	existing.Settings = newSettings
	existing.LoadedSettings = settings

	setUsers(existing.LoadedSettings, existingUsers)
	existing.Settings = existing.LoadedSettings.Base64Encode()

	if !settings.IsDemo() {
		e := s.buildEnvironment(existing)
		validationErr := e.Validate()
		if validationErr != nil {
			return nil, &model.AppError{Message: validationErr.Error(), Code: model.InvalidProviderConfiguration, Status: http.StatusBadRequest}
		}
	}

	tx := s.DB.Begin()
	if tx.Error != nil {
		return nil, &model.AppError{Message: tx.Error.Error(), Code: model.InternalServerError, Status: http.StatusInternalServerError}
	}

	err := tx.Save(&existing).Error
	if err != nil {
		tx.Rollback()
		return nil, &model.AppError{Message: err.Error(), Code: model.InternalServerError, Status: http.StatusInternalServerError}
	}

	err = tx.Where("project_id = ?", existing.ID).Delete(&model.ProjectACL{}).Error
	if err != nil {
		tx.Rollback()
		return nil, &model.AppError{Message: err.Error(), Code: model.InternalServerError, Status: http.StatusInternalServerError}
	}

	for _, a := range existingUsers {
		err = tx.Exec("INSERT into project_acls(project_id, user_id, role) values(?, (SELECT id from users WHERE email = ?), ?)", a.ProjectID, a.UserEmail, a.Role).Error
		if err != nil {
			tx.Rollback()
			return nil, &model.AppError{Message: err.Error(), Code: model.InternalServerError, Status: http.StatusInternalServerError}
		}
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return nil, &model.AppError{Message: err.Error(), Code: model.InternalServerError, Status: http.StatusInternalServerError}
	}

	invitedUsers := findInvitedUsers(previousUsers, append(existing.LoadedSettings.Administrators, existing.LoadedSettings.Users...))
	s.sendProjectInvite(invitedUsers, existing.Name)
	return pendingUsers, nil
}

// UpdateProjectGithubLink Updates the link between the project and a github installation
func (s *Server) UpdateProjectGithubLink(loggedUser *model.User, existing *model.Project) *model.AppError {
	new := &model.Github{LinkedBy: loggedUser.Email}
	ghInstallationID, appErr := validateGithubScope(existing.LoadedSettings.Github, new, loggedUser)
	if appErr != nil {
		return appErr
	}

	existing.LoadedSettings.Github = new
	existing.GHInstallationID = ghInstallationID
	existing.Settings = existing.LoadedSettings.Base64Encode()

	err := s.DB.Save(&existing).Error
	if err != nil {
		return &model.AppError{Message: err.Error(), Code: model.InternalServerError, Status: http.StatusInternalServerError}
	}
	return nil
}

// RemoveProjectGithubLink removes the link between the project and a github installation and
// deletes all gh_repo_links for the services in the project.
func (s *Server) RemoveProjectGithubLink(existing *model.Project, tx *gorm.DB) error {
	if existing.LoadedSettings == nil {
		existing.LoadedSettings, _ = model.ParseProjectSettings(existing.Settings)
	}

	existing.LoadedSettings.Github = nil
	existing.GHInstallationID = ""
	existing.Settings = existing.LoadedSettings.Base64Encode()

	if tx == nil {
		tx = s.DB.Begin()
		if tx.Error != nil {
			return errors.Wrap(tx.Error, "couldn't start transaction")
		}
	}

	err := tx.Save(&existing).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrapf(err, "couldn't update project-%s", existing.ID)
	}

	var services []model.Service
	err = tx.Where(model.Service{ProjectID: existing.ID}).Find(&services).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrapf(err, "couldn't get services for project-%s", existing.ID)
	}

	toRemoveLink := []string{}
	ghRepoLinkIDs := []string{}

	for _, svc := range services {
		if svc.GHRepoLinkID != "" {
			toRemoveLink = append(toRemoveLink, svc.ID)
			ghRepoLinkIDs = append(ghRepoLinkIDs, svc.GHRepoLinkID)
		}
	}

	if len(toRemoveLink) > 0 {
		err = tx.Model(model.Service{}).Where("id IN (?)", toRemoveLink).Updates(model.Service{GHRepoLinkID: ""}).Error
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "couldn't remove gh_repo_link from services")
		}

		err = tx.Model(model.GHRepoLink{}).Where("id IN (?)", ghRepoLinkIDs).Delete(model.GHRepoLink{}).Error
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "couldn't delete gh_repo_links")
		}
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "couldn't commit transaction for RemoveProjectGithubLink")
	}

	return nil
}

func findInvitedUsers(originalList []string, updatedList []string) []string {

	invitedUsers := []string{}

	for _, u := range updatedList {
		invitedUser := true
		for _, o := range originalList {
			if u == o {
				invitedUser = false
				break
			}
		}

		if invitedUser {
			invitedUsers = append(invitedUsers, u)
		}
	}

	return invitedUsers
}

// DeleteProject deletes the project or throws an error
func (s *Server) DeleteProject(projectID string, force bool) *model.AppError {

	if !force {
		count, err := s.getActiveServicesCount(projectID)

		if err != nil {
			return &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: err.Error()}
		}

		if count != 0 {
			return &model.AppError{Status: http.StatusConflict, Code: model.ProjectNotEmpty}
		}
	}

	result := s.DB.Where("id = ?", projectID).Delete(&model.Project{})

	if result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}

		return &model.AppError{Status: http.StatusInternalServerError, Code: model.InternalServerError, Message: result.Error.Error()}
	}

	return nil
}

func (s *Server) validateACLs(acls []model.ProjectACL) ([]model.ProjectACL, []model.ProjectACL, *model.AppError) {

	pendingUsers := make([]model.ProjectACL, 0)
	currentUsers := make([]model.ProjectACL, 0)

	for _, acl := range acls {

		u, err := s.GetUser(acl.UserEmail)
		if err != nil {
			if err.Code == model.EntityNotFound {
				pendingUsers = append(pendingUsers, acl)
			} else {
				return nil, nil, err
			}
		} else {
			if u.Verified {
				currentUsers = append(currentUsers, acl)
			} else {
				// When sending the invite, we can deduplicate if we already invited the user
				pendingUsers = append(pendingUsers, acl)
			}
		}
	}

	pendingUsers = uniqueUsers(pendingUsers)
	currentUsers = uniqueUsers(currentUsers)
	return currentUsers, pendingUsers, nil
}

func validateGithubScope(existing *model.Github, new *model.Github, loggedUser *model.User) (string, *model.AppError) {
	if new == nil {
		return "", nil
	}

	if existing != nil {
		if existing.LinkedBy == new.LinkedBy {
			return "", nil
		}

		if new.LinkedBy != loggedUser.Email {
			return "", &model.AppError{
				Message: "invalid github scope",
				Code:    model.InvalidGithubScope,
				Status:  http.StatusBadRequest}
		}
	}

	if loggedUser.GHInstallation == nil {
		return "", &model.AppError{
			Message: fmt.Sprintf("user-%s is not linked to github", loggedUser.ID),
			Code:    model.InvalidGithubScope,
			Status:  http.StatusBadRequest}
	}

	return loggedUser.GHInstallation.ID, nil
}

func setUsers(settings *model.ProjectSettings, existingUsers []model.ProjectACL) {
	settings.Users = make([]string, 0)
	settings.Administrators = make([]string, 0)
	for i := range existingUsers {
		if existingUsers[i].Role == model.ProjectRoleAdmin {
			settings.Administrators = append(settings.Administrators, existingUsers[i].UserEmail)
		} else {
			settings.Users = append(settings.Users, existingUsers[i].UserEmail)
		}
	}
}

func (s *Server) sendProjectInvite(users []string, projectName string) {
	url := config.GetBaseURL()
	for _, u := range users {
		err := s.Email.sendProjectInviteEmail(u, projectName, url)
		if err != nil {
			log.Printf("Failed to send email: %s", err.Error())
		}
	}
}

func uniqueUsers(users []model.ProjectACL) []model.ProjectACL {
	keys := make(map[string]model.ProjectRole)
	list := []model.ProjectACL{}

	for _, entry := range users {
		_, value := keys[entry.UserEmail]
		if !value {
			keys[entry.UserEmail] = entry.Role
			list = append(list, entry)
		} else {
			if entry.Role == model.ProjectRoleAdmin {
				for i := range list {
					if list[i].UserEmail == entry.UserEmail {
						list[i] = entry
					}
				}
			}
		}
	}

	return list
}

func checkForUniqueDNSError(err error) bool {
	return strings.Contains(strings.ToUpper(err.Error()), "UNIQUE CONSTRAINT")
}
