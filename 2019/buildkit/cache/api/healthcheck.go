package api

import (
	"net/http"

	"bitbucket.org/okteto/okteto/backend/logger"

	restful "github.com/emicklei/go-restful"
	"github.com/pkg/errors"
)

// Healthcheck type for the response
type Healthcheck struct {
	Code    int
	Message string
}

func (a *API) getHealthcheck(request *restful.Request, response *restful.Response) {
	if err := a.app.PingDB(); err != nil {
		logger.Error(errors.Wrap(err, "healthcheck failed"))
		response.WriteHeaderAndEntity(http.StatusInternalServerError, Healthcheck{Code: http.StatusInternalServerError, Message: "oops"})
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, Healthcheck{Code: http.StatusOK, Message: "a-ok"})
}
