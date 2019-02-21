package api

import (
	"log"
	"net/http"
	"strings"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"

	"bitbucket.org/okteto/okteto/backend/config"
	restful "github.com/emicklei/go-restful"
	"github.com/futurenda/google-auth-id-token-verifier"
	"github.com/pkg/errors"
)

const authChallengeHeader = "Bearer realm=Protected Area"

//GoogleAuthRequest is the request to authenticate that comes from google
type GoogleAuthRequest struct {
	Token    string `json:"token"`
	Mixpanel string `json:"mixpanel"`
}

//BasicAuthRequest is the request used by the tests
type BasicAuthRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Secret string `json:"secret"`
}

//AuthResponse is the response send back after authenticating a user successfully
type AuthResponse struct {
	Name           string          `json:"name,omitempty"`
	Picture        string          `json:"pictureURL,omitempty"`
	Email          string          `json:"email"`
	UserID         string          `json:"userID"`
	Token          string          `json:"token"`
	Verified       bool            `json:"verified"`
	Github         bool            `json:"github"`
	Projects       []model.Project `json:"projects"`
	Mixpanel       string          `json:"mixpanel"`
	DefaultCluster string          `json:"defaultCluster"`
}

func (a *API) authBasic(request *restful.Request, response *restful.Response) {
	ar := BasicAuthRequest{}
	err := request.ReadEntity(&ar)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read the request body"))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	if ar.Email == "" {
		log.Print("email was empty")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	if ar.Secret == "" {
		log.Print("secret was empty")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	if ar.Secret != config.GetAuthSecret() {
		log.Print("bad secret")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	u, appErr := a.app.GetOrCreateUser(ar.Email, true, "")
	if appErr != nil {
		response.WriteHeader(400)
		logger.Error(errors.Wrapf(appErr, "failed to retrieve or create user-%s", ar.Email))
		return
	}

	arp := AuthResponse{
		Name:           ar.Name,
		Email:          ar.Email,
		Token:          u.Token,
		Verified:       u.Verified,
		UserID:         u.ID,
		Github:         u.GHInstallation != nil,
		DefaultCluster: config.GetClusterName(),
	}

	projects, appErr := a.app.GetProjects(u)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to get projects for user-%s", u.ID))
	}

	arp.Projects = projects

	response.WriteEntity(arp)
}

func (a *API) authGoogle(request *restful.Request, response *restful.Response) {
	ar := GoogleAuthRequest{}
	err := request.ReadEntity(&ar)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to read the token body"))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	if ar.Token == "" {
		logger.Error(errors.New("google token was not in the request"))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	gu, err := getInformationFromGoogle(ar.Token)
	if err != nil {
		logger.Error(err)
		response.WriteHeader(400)
		return
	}

	u, appErr := a.app.GetOrCreateUser(gu.Email, true, ar.Mixpanel)
	if appErr != nil {
		response.WriteHeader(appErr.Status)
		logger.Error(errors.Wrapf(appErr, "failed to retrieve or create user-%s", gu.Email))
		return
	}

	if !u.Verified {
		// TODO: Remove once we have a privacy screen
		a.app.VerifyUser(u)
	}

	if u.Mixpanel == "" && ar.Mixpanel != "" {
		if err := a.app.LinkToMixpanel(u, ar.Mixpanel); err != nil {
			logger.Error(err)
		}

		u.Mixpanel = ar.Mixpanel
	}

	projects, appErr := a.app.GetProjects(u)
	if appErr != nil {
		logger.Error(errors.Wrapf(appErr, "failed to retrieve projects for uid-%s", u.ID))
	}

	gu.Projects = projects
	gu.Token = u.Token
	gu.Verified = u.Verified
	gu.UserID = u.ID
	gu.Github = u.GHInstallation != nil
	gu.DefaultCluster = config.GetClusterName()
	gu.Mixpanel = u.Mixpanel
	response.WriteEntity(gu)
}

func getInformationFromGoogle(idToken string) (*AuthResponse, error) {
	v := googleAuthIDTokenVerifier.Verifier{}
	aud := config.GetGoogleAuthID()
	err := v.VerifyIDToken(idToken, []string{aud})

	if err != nil {
		return nil, errors.Wrap(err, "failed to validate the token information from google")
	}

	claimSet, err := googleAuthIDTokenVerifier.Decode(idToken)

	if err != nil {
		return nil, errors.Wrap(err, "failed to decode the token information from google")
	}

	a := AuthResponse{
		Email:   claimSet.Email,
		Name:    claimSet.Name,
		Picture: claimSet.Picture,
	}

	return &a, nil
}

func (a *API) apiTokenAuthentication(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {

	var token string
	token = req.Request.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")

	if token == "" {
		resp.AddHeader("WWW-Authenticate", authChallengeHeader)
		appErr := &model.AppError{Status: http.StatusUnauthorized, Code: model.MissingToken}
		resp.WriteHeaderAndEntity(appErr.Status, appErr)
		logger.Info("api call denied due to missing bearer token: %s", req.Request.RequestURI)
		return
	}

	u, err := a.app.GetUserByToken(token)

	if err != nil {
		resp.AddHeader("WWW-Authenticate", authChallengeHeader)
		appErr := &model.AppError{Status: http.StatusUnauthorized, Code: model.InvalidToken}
		resp.WriteHeaderAndEntity(appErr.Status, appErr)
		logger.Info("api call denied due to bearer token not matching a registered user: %s", token[:5])
		return
	}

	projectID := req.PathParameters()["project-id"]
	if projectID != "" {
		p, appErr := a.app.GetProject(projectID, u.ID)

		if appErr != nil {
			if appErr.Code == model.EntityNotFound {
				resp.AddHeader("WWW-Authenticate", authChallengeHeader)
				resp.WriteHeader(http.StatusForbidden)
				log.Printf("api call denied due user-%s trying to access project-%s", u.ID, projectID)

			} else {
				logger.Error(errors.Wrap(appErr, "failed to get project during auth"))
				resp.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		if p.Settings != "" {
			s, appErr := model.ParseProjectSettings(p.Settings)
			if err != nil {
				logger.Error(errors.Wrapf(appErr, "project-%s has malformed settings", p.ID))
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}

			p.LoadedSettings = s
		}

		setRequestedProject(req, p)
	}

	setAuthenticatedUser(req, u)
	chain.ProcessFilter(req, resp)
}

func getAuthenticatedUser(req *restful.Request) *model.User {
	u := req.Attribute("user")
	return u.(*model.User)
}

func setAuthenticatedUser(req *restful.Request, u *model.User) {
	req.SetAttribute("user", u)
}

func setRequestedProject(req *restful.Request, p *model.Project) {
	req.SetAttribute("project", p)
}

func getRequestedProject(req *restful.Request) *model.Project {
	p := req.Attribute("project")
	return p.(*model.Project)
}
