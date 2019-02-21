package app

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"bitbucket.org/okteto/okteto/backend/logger"
	"github.com/pkg/errors"

	"bitbucket.org/okteto/okteto/backend/config"
	"bitbucket.org/okteto/okteto/backend/model"
)

var internalUsers = []string{
	"rberrelleza@gmail.com",
	"rlamana@gmail.com",
	"pchico83@gmail.com",
	"@okteto.com",
}

func (s *Server) newAccountNotification(u *model.User) {
	if isInternalUser(u.Email) {
		return
	}

	sendSlackNotification(fmt.Sprintf(`{"text":"%s joined okteto!"}`, u.Email))
}

func (s *Server) newProjectNotification(u *model.User) {
	if isInternalUser(u.Email) {
		return
	}

	sendSlackNotification(fmt.Sprintf(`{"text":"%s created a new project."}`, u.Email))
}

func (s *Server) newServiceNotification(u *model.User) {
	if isInternalUser(u.Email) {
		return
	}

	sendSlackNotification(fmt.Sprintf(`{"text":"%s created a new service."}`, u.Email))
}

func (s *Server) deployedServiceNotification(u *model.User) {
	if isInternalUser(u.Email) {
		return
	}

	sendSlackNotification(fmt.Sprintf(`{"text":"%s deployed a new service."}`, u.Email))
}

func (s *Server) devDeployedServiceNotification(u *model.User) {
	if isInternalUser(u.Email) {
		return
	}

	sendSlackNotification(fmt.Sprintf(`{"text":"%s enabled dev mode."}`, u.Email))
}

func sendSlackNotification(payload string) {
	webhookURL := config.GetSlackWebhook()
	if webhookURL == "" {
		return
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to send notification to the slack channel"))
	}
}

func isInternalUser(email string) bool {
	for _, e := range internalUsers {
		if strings.Contains(email, e) {
			return true
		}
	}

	return false
}
