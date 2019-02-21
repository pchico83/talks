package api

import (
	"net/http"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"
	restful "github.com/emicklei/go-restful"
	"github.com/pkg/errors"
)

// GHInstallationClaimRequest is the request to link a github installation to an okteto user
type GHInstallationClaimRequest struct {
	Installation int
}

func (a *API) getUser(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)

	projects, appErr := a.app.GetProjects(u)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get projects for user-%s", u.ID))
	}

	authResponse := AuthResponse{
		Email:    u.Email,
		Token:    u.Token,
		UserID:   u.ID,
		Verified: u.Verified,
		Github:   u.GHInstallation != nil,
		Mixpanel: u.Mixpanel,
		Projects: projects,
	}

	response.WriteEntity(authResponse)
}

func (a *API) inviteUser(request *restful.Request, response *restful.Response) {
	r := model.InviteUserRequest{}
	err := request.ReadEntity(&r)

	if err != nil {
		appErr := &model.AppError{Status: http.StatusBadRequest, Code: model.InvalidJSON}
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	appErr := r.Validate()
	if appErr != nil {
		response.WriteHeaderAndEntity(http.StatusBadRequest, appErr)
		return
	}

	authenticated := getAuthenticatedUser(request)
	appErr = a.app.InviteUser(authenticated, r.Email, r.Project, r.Role)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to invite user %s", r.Email))
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (a *API) verifyUser(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)
	appErr := a.app.VerifyUser(u)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to verify user-%s", u.ID))
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		response.WriteHeader(http.StatusNoContent)
	}
}

func (a *API) deleteUser(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)
	appErr := a.app.DeleteUser(u)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to delete user-%s", u.ID))
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		response.WriteHeader(http.StatusNoContent)
	}
}

func (a *API) claimGHInstallation(request *restful.Request, response *restful.Response) {
	u := getAuthenticatedUser(request)
	l := GHInstallationClaimRequest{}

	err := request.ReadEntity(&l)
	if err != nil {
		appErr := &model.AppError{Status: http.StatusBadRequest, Code: model.InvalidJSON}
		response.WriteHeaderAndEntity(appErr.Status, appErr)
		return
	}

	err = a.app.ClaimGHInstallation(u.ID, l.Installation)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to link ghinstallation"))
		if err == model.ErrNotFound {
			response.WriteHeader(http.StatusBadRequest)
		} else {
			response.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	response.WriteHeader(http.StatusOK)
}
