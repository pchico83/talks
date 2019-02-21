package api

import (
	"log"
	"net/http"
	"strconv"

	"bitbucket.org/okteto/okteto/backend/config"

	"github.com/emicklei/go-restful"
	"github.com/pkg/errors"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (a *API) getProjects(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)
	projects, appErr := a.app.GetProjects(u)

	if appErr != nil {
		response.WriteHeader(appErr.Status)
		return
	}

	for i := range projects {
		sanitizeOutput(&projects[i], u)
	}

	response.WriteEntity(projects)
}

func (a *API) getProject(request *restful.Request, response *restful.Response) {
	p := getRequestedProject(request)
	u := getAuthenticatedUser(request)
	sanitizeOutput(p, u)
	response.WriteEntity(p)
}

func sanitizeOutput(p *model.Project, u *model.User) {
	if !config.UseInClusterConfig() {
		p.IsFree = p.LoadedSettings.Provider.IsFreeTierProvider()
	}

	if p.Role != model.ProjectRoleAdmin {
		p.Settings = ""
	}

	return
}

func (a *API) createProject(request *restful.Request, response *restful.Response) {
	d := &model.Project{}
	err := request.ReadEntity(&d)
	if err != nil {
		appErr := &model.AppError{Status: http.StatusBadRequest, Code: model.InvalidJSON}
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	u := getAuthenticatedUser(request)
	p, appErr := a.app.CreateProject(d, u)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to create new project"))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	newProject, appErr := a.app.GetProject(p, u.ID)
	if err != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get project-%s", p))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	sanitizeOutput(newProject, u)
	response.WriteEntity(newProject)
}

func (a *API) updateProject(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)
	p := getRequestedProject(request)

	if !p.LoadedSettings.IsAdmin(&u.Email) {
		response.WriteHeader(http.StatusForbidden)
		return
	}

	newValues := &model.Project{}
	err := request.ReadEntity(&newValues)

	if err != nil {
		appErr := &model.AppError{Status: http.StatusBadRequest, Code: model.InvalidJSON}
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	pending, appErr := a.app.UpdateProjectSettings(u, p, newValues.Settings)

	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to update project-%s", p.ID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	updatedProject, appErr := a.app.GetProject(p.ID, u.ID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to read updated project-%s", p.ID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	if len(pending) > 0 {
		updatedProject.PendingUsers = pending
	}

	sanitizeOutput(updatedProject, u)
	response.WriteEntity(updatedProject)
}

func (a *API) updateProjectGithubLink(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)
	p := getRequestedProject(request)

	if !p.LoadedSettings.IsAdmin(&u.Email) {
		response.WriteHeader(http.StatusForbidden)
		return
	}

	appErr := a.app.UpdateProjectGithubLink(u, p)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to update gh_link for project-%s", p.ID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	updatedProject, appErr := a.app.GetProject(p.ID, u.ID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to read updated project-%s", p.ID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	response.WriteEntity(updatedProject)
}

func (a *API) removeProjectGithubLink(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)
	p := getRequestedProject(request)

	if !p.LoadedSettings.IsAdmin(&u.Email) {
		response.WriteHeader(http.StatusForbidden)
		return
	}

	err := a.app.RemoveProjectGithubLink(p, nil)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to delete gh_link for project-%s", p.ID))
		response.WriteHeaderAndEntity(http.StatusInternalServerError, err)
		return
	}

	updatedProject, appErr := a.app.GetProject(p.ID, u.ID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to read updated project-%s", p.ID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	sanitizeOutput(updatedProject, u)
	response.WriteEntity(updatedProject)
}

func (a *API) deleteProject(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)
	p := getRequestedProject(request)
	force, _ := strconv.ParseBool(request.QueryParameter("force"))

	if !p.LoadedSettings.IsAdmin(&u.Email) {
		response.WriteHeader(http.StatusForbidden)
		return
	}

	appErr := a.app.DeleteProject(p.ID, force)

	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to delete project-%s", p.ID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (a *API) startWebsocket(request *restful.Request, response *restful.Response) {
	conn, err := upgrader.Upgrade(response, request.Request, response.Header())
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to start a websocket"))
		return
	}

	projectID := request.PathParameters()["project-id"]

	ws := a.app.Hub.StartNewClient(conn, projectID, a.app.GetUserByToken)
	log.Printf("starting ws-%s", ws)
}

func (a *API) getGHRepositories(request *restful.Request, response *restful.Response) {
	p := getRequestedProject(request)

	repositories, err := a.app.GetGHRepositories(p)
	if err != nil {
		if err == model.ErrProjectNotLinkedToGithub {
			logger.Error(errors.Wrapf(err, "project-%s is not linked with gh", p.ID))
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		if isNotFoundErr(err) {
			// bad request because this covers the case when we can't find the installation
			response.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.Error(errors.Wrapf(err, "failed to read repositories of project-%s", p.ID))
		response.WriteHeader(http.StatusInternalServerError)
		return

	}

	out := make([]GHRepository, len(repositories))
	for i := range repositories {
		out[i] = GHRepository{
			Name:  repositories[i].GetName(),
			Owner: repositories[i].GetOwner().GetLogin(),
			URL:   repositories[i].GetURL(),
		}
	}

	response.WriteEntity(out)
}

func isNotFoundErr(err error) bool {
	cause := errors.Cause(err)
	return cause == model.ErrNotFound
}
