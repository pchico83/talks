package api

import (
	"log"
	"net/http"

	"bitbucket.org/okteto/okteto/backend/app"
	"bitbucket.org/okteto/okteto/backend/logger"
	"github.com/pkg/errors"

	restful "github.com/emicklei/go-restful"
)

const (
	// integration_installation is the legacy event. It was superceded by `installation`
	githubEventHeader = "X-Github-Event"
)

// GHRepository contains info about a github repo. It can be used both to send info and read link requests
type GHRepository struct {
	Owner    string `json:"owner,omitempty"`
	Name     string `json:"name,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Manifest string `json:"manifest,omitempty"`
	URL      string `json:"url,omitempty"`
}

func (a *API) ghWebhook(request *restful.Request, response *restful.Response) {
	event := request.HeaderParameter("X-Github-Event")
	logger.Info("ghWebhook called on %s for event '%s'", request.Request.RequestURI, event)

	if !app.IsGithubEventSupported(event) {
		log.Printf("unknown github event %s", event)
		return
	}

	payload := &app.GHWebhookPayload{}
	err := request.ReadEntity(&payload)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to load json from github webhook"))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	payload.Event = event
	a.app.QueueGithubEvent(payload)

}
