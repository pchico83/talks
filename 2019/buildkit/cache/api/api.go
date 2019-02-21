package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bitbucket.org/okteto/okteto/backend/app"
	"bitbucket.org/okteto/okteto/backend/logger"

	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/emicklei/go-restful"
	"github.com/pkg/errors"
)

//API contains the app instance and the container that executes the api calls
type API struct {
	app       *app.Server
	Container *restful.Container
}

//Init registers the api handlers
func Init(s *app.Server) http.Handler {
	a := newAPI(s)
	handler := buildMetricsHandler(a.Container)
	return handler
}

func newAPI(s *app.Server) *API {
	a := &API{app: s}
	a.initRouter()
	a.registerServices()
	return a
}

func (a *API) initRouter() {
	a.Container = restful.DefaultContainer
	a.Container.DoNotRecover(false)
	a.Container.RecoverHandler(logStackOnRecover)
	a.Container.Router(restful.CurlyRouter{})

}

func (a *API) registerServices() {
	a.Container.Add(a.registerHealthcheckAPI())
	a.Container.Add(a.registerProjectsAPI())
	a.Container.Add(a.registerEventsAPI())
	a.Container.Add(a.registerUsersAPI())
	a.Container.Add(a.registerAuthAPI())
	a.Container.Add(a.registerGithubAPI())
	a.Container.Add(a.registerConfigAPI())

	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"x-okteto"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "PATCH", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      a.Container,
	}
	a.Container.Filter(durationFilter)
	a.Container.Filter(cors.Filter)
	a.Container.Filter(a.Container.OPTIONSFilter)
}

func (a *API) registerHealthcheckAPI() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/api/v1/healthcheck").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").To(a.getHealthcheck).
		Returns(http.StatusOK, "OK", nil))

	return ws
}

func (a *API) registerConfigAPI() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/api/v1/config").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").To(a.getConfig).
		Returns(http.StatusOK, "OK", PublicConfig{}))

	return ws
}

func (a *API) registerProjectsAPI() *restful.WebService {
	ws := new(restful.WebService).Filter(a.apiTokenAuthentication)

	ws.Path("/api/v1/projects").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").To(a.getProjects).
		Writes([]model.Project{}).
		Returns(200, "OK", []model.Project{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{project-id}").To(a.getProject).
		Writes(model.Project{}).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(200, "OK", model.Project{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{project-id}").To(a.deleteProject).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(204, "OK", nil).
		Returns(404, "Not Found", nil).
		Returns(409, "Conflict", nil))

	ws.Route(ws.POST("").To(a.createProject).
		Writes(model.Project{}).
		Reads(model.Project{}).
		Returns(200, "OK", model.Project{}).
		Returns(400, "Bad Request", nil))

	ws.Route(ws.PUT("/{project-id}").To(a.updateProject).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(200, "OK", model.Project{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.PUT("/{project-id}/github").To(a.updateProjectGithubLink).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(200, "OK", model.Project{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{project-id}/github").To(a.removeProjectGithubLink).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(200, "OK", model.Project{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{project-id}/github/repositories").To(a.getGHRepositories).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(200, "OK", model.Project{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{project-id}/services/{service-id}").To(a.getService).
		Writes(model.Service{}).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Returns(200, "OK", model.Service{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.POST("/{project-id}/services/{service-id}/deploy").To(a.startService).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Returns(200, "OK", model.Service{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{project-id}/services/{service-id}/activities/{activity-id}/logs").To(a.getActivityLogs).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Param(ws.PathParameter("activity-id", "identifier of the activity").DataType("string")).
		Returns(200, "OK", []model.ActivityLog{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{project-id}/services/{service-id}").To(a.deleteService).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Returns(204, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.PUT("/{project-id}/services/{service-id}").To(a.updateService).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Returns(200, "OK", model.Service{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.POST("/{project-id}/services").To(a.createService).
		Writes(model.Service{}).
		Reads(model.Service{}).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(200, "OK", model.Service{}).
		Returns(400, "Bad Request", nil))

	ws.Route(ws.PUT("/{project-id}/services/{service-id}/github").To(a.linkGHRepositoryToService).
		Reads(GHRepository{}).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Returns(204, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{project-id}/services/{service-id}/github").To(a.unlinkGHRepositoryToService).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Returns(204, "OK", nil).
		Returns(404, "Not Found", nil))

	ws.Route(ws.POST("/{project-id}/services/{service-id}/dev").To(a.enableDevMode).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Returns(204, "OK", model.Service{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{project-id}/credentials").To(a.credentials).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(200, "OK", app.Credentials{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.DELETE("/{project-id}/services/{service-id}/dev").To(a.startService).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.PathParameter("service-id", "identifier of the service").DataType("string")).
		Returns(204, "OK", model.Service{}).
		Returns(404, "Not Found", nil))

	ws.Route(ws.GET("/{project-id}/services").To(a.getServices).
		Writes([]model.Service{}).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Param(ws.QueryParameter("include-deleted", "include deleted services").DataType("bool")).
		Returns(200, "OK", []model.Service{}))

	return ws
}

func (a *API) registerEventsAPI() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/api/v1/events").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/projects/{project-id}").To(a.startWebsocket).
		Param(ws.PathParameter("project-id", "identifier of the project").DataType("string")).
		Returns(200, "OK", nil).
		Returns(400, "Bad Request", nil))
	return ws
}

func (a *API) registerAuthAPI() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/api/v1/auth").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/google").To(a.authGoogle).
		Writes(AuthResponse{}).
		Reads(GoogleAuthRequest{}).
		Returns(200, "OK", AuthResponse{}).
		Returns(400, "Bad Request", nil).
		Returns(401, "Unauthorized", nil))

	ws.Route(ws.POST("/basic").To(a.authBasic).
		Writes(AuthResponse{}).
		Reads(BasicAuthRequest{}).
		Returns(200, "OK", AuthResponse{}).
		Returns(400, "Bad Request", nil).
		Returns(401, "Unauthorized", nil))

	return ws
}

func (a *API) registerUsersAPI() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/api/v1/users").Filter(a.apiTokenAuthentication).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").To(a.getUser).
		Writes(model.User{}).
		Returns(200, "OK", nil).
		Returns(400, "Bad Request", nil))

	ws.Route(ws.POST("/invite").To(a.inviteUser).
		Reads(model.InviteUserRequest{}).
		Returns(200, "OK", nil).
		Returns(400, "Bad Request", nil))

	ws.Route(ws.POST("/verify").To(a.verifyUser).
		Returns(204, "OK", nil))

	ws.Route(ws.PUT("/github").To(a.claimGHInstallation).
		Reads(GHInstallationClaimRequest{}).
		Returns(204, "OK", nil))

	ws.Route(ws.DELETE("").To(a.deleteUser).
		Returns(204, "OK", nil))

	return ws
}

func (a *API) registerGithubAPI() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/api/v1/github").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/events").To(a.ghWebhook).
		Returns(204, "OK", nil).
		Returns(400, "Bad Request", nil))

	return ws
}

func durationFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()
	chain.ProcessFilter(req, resp)
	duration.WithLabelValues(strconv.Itoa(resp.StatusCode()), req.SelectedRoutePath()).Observe(time.Since(start).Seconds())
	return
}

func logStackOnRecover(panicReason interface{}, httpWriter http.ResponseWriter) {

	if r, ok := panicReason.(error); ok {
		logger.Error(errors.Wrap(r, "[api] recover from panic situation"))
	} else {
		logger.Error(errors.Wrap(fmt.Errorf("%s", panicReason), "[api] recover from panic situation"))
	}

	httpWriter.WriteHeader(http.StatusInternalServerError)
}
