package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/pkg/errors"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"
)

func (a *API) getService(request *restful.Request, response *restful.Response) {
	serviceID := request.PathParameter("service-id")
	project := getRequestedProject(request)
	service, appErr := a.app.GetServiceAndActivities(project, serviceID)

	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get service-%s", serviceID))
		response.WriteHeader(appErr.Status)
		return
	}

	response.WriteEntity(service)
}

func (a *API) getServices(request *restful.Request, response *restful.Response) {
	project := getRequestedProject(request)
	includeDeleted, _ := strconv.ParseBool(request.QueryParameter("include-deleted"))
	services, appErr := a.app.GetServices(project.ID, includeDeleted)

	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get services"))
		response.WriteHeader(appErr.Status)
		return
	}

	response.WriteEntity(services)
}

func (a *API) createService(request *restful.Request, response *restful.Response) {
	svc := &model.Service{}
	if err := request.ReadEntity(&svc); err != nil {
		appErr := &model.AppError{Status: http.StatusBadRequest, Code: model.InvalidJSON}
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	p := getRequestedProject(request)
	u := getAuthenticatedUser(request)
	svc.ProjectID = p.ID
	svc.CreatedBy = u.ID

	svc.IsDemo = p.IsFree

	newService, appErr := a.app.CreateService(p, svc, u)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to create service"))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	newService, appErr = a.app.GetServiceAndActivities(p, newService.ID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get created service-%s", newService.ID))
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.WriteEntity(newService)
}

func (a *API) startService(request *restful.Request, response *restful.Response) {
	serviceID := request.PathParameter("service-id")
	project := getRequestedProject(request)
	user := getAuthenticatedUser(request)
	log.Printf("starting service-%s", serviceID)

	appErr := a.app.StartService(project, serviceID, user)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get start service-%s", serviceID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	service, appErr := a.app.GetServiceAndActivities(project, serviceID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get started service-%s", serviceID))
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.WriteEntity(service)
}

func (a *API) enableDevMode(request *restful.Request, response *restful.Response) {
	serviceID := request.PathParameter("service-id")
	project := getRequestedProject(request)
	user := getAuthenticatedUser(request)
	log.Printf("enabling dev mode for service-%s", serviceID)

	appErr := a.app.EnableDevMode(project, serviceID, user)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to enable dev mode for service-%s", serviceID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	service, appErr := a.app.GetServiceAndActivities(project, serviceID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get service-%s", serviceID))
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.WriteEntity(service)
}

func (a *API) credentials(request *restful.Request, response *restful.Response) {
	project := getRequestedProject(request)
	log.Printf("getting credentials for project-%s", project.ID)

	credentials, appErr := a.app.Credentials(project)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get credentials for project-%s", project.ID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}
	response.WriteEntity(credentials)
}

func (a *API) updateService(request *restful.Request, response *restful.Response) {
	serviceID := request.PathParameter("service-id")
	project := getRequestedProject(request)
	user := getAuthenticatedUser(request)
	d := &model.Service{}
	err := request.ReadEntity(&d)
	if err != nil {
		appErr := &model.AppError{Status: http.StatusBadRequest, Code: model.InvalidJSON}
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	d.ID = serviceID
	appErr := a.app.UpdateManifest(project.ID, serviceID, d.Manifest, user.ID, "Manifest updated manually")
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to update service-%s", serviceID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	service, appErr := a.app.GetServiceAndActivities(project, serviceID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get updated service-%s", serviceID))
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.WriteEntity(service)
}

func (a *API) deleteService(request *restful.Request, response *restful.Response) {
	project := getRequestedProject(request)
	serviceID := request.PathParameter("service-id")
	appErr := a.app.DeleteService(project, serviceID, getAuthenticatedUser(request))

	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to delete service-%s", serviceID))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	service, appErr := a.app.GetServiceAndActivities(project, serviceID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get deleted service-%s", serviceID))
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.WriteEntity(service)
}

func (a *API) getActivityLogs(request *restful.Request, response *restful.Response) {
	activityID := request.PathParameter("activity-id")
	serviceID := request.PathParameter("service-id")
	project := getRequestedProject(request)
	logs, appErr := a.app.GetActivityLogs(project, serviceID, activityID)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get actity logs for activity-%s", activityID))
		response.WriteHeader(http.StatusNotFound)
		return
	}

	response.WriteEntity(logs)
}

func (a *API) linkGHRepositoryToService(request *restful.Request, response *restful.Response) {
	serviceID := request.PathParameter("service-id")
	project := getRequestedProject(request)

	if project.GHInstallationID == "" {
		response.WriteHeader(http.StatusBadRequest)
		logger.Error(fmt.Errorf("project-%s is not linked to github", project.ID))
		return
	}

	r := &GHRepository{}
	err := request.ReadEntity(&r)
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		logger.Error(errors.Wrap(err, "failed to read json from request"))
		return
	}

	err = a.app.LinkGHRepositoryToService(project, serviceID, r.Owner, r.Name, r.Branch, r.Manifest)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to link service-%s", serviceID))
	}
}

func (a *API) unlinkGHRepositoryToService(request *restful.Request, response *restful.Response) {
	serviceID := request.PathParameter("service-id")
	project := getRequestedProject(request)

	err := a.app.UnlinkGHRepositoryToService(project, serviceID)
	if err != nil {
		logger.Info(err.Error())
		if isNotFoundErr(err) {
			response.WriteHeader(http.StatusNotFound)
			return
		}

		logger.Error(errors.Wrapf(err, "failed to unlink service-%s", serviceID))
		response.WriteHeader(http.StatusInternalServerError)
	}
}
