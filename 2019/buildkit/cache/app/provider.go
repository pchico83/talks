package app

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/providers"
	"github.com/pkg/errors"
)

const providerTimeout = 15 * time.Minute

func (s *Server) devDeploy(d *model.Service, project *model.Project, activityID string) error {
	return s.callProvider(d, project, activityID, providers.DevDeploy)
}

func (s *Server) deploy(d *model.Service, project *model.Project, activityID string) error {
	return s.callProvider(d, project, activityID, providers.Deploy)
}

func (s *Server) destroy(d *model.Service, project *model.Project, activityID string) error {
	return s.callProvider(d, project, activityID, providers.Destroy)
}

func (s *Server) callProvider(d *model.Service, project *model.Project, activityID string, f func(*model.Service, *model.Environment, *log.Logger) error) error {
	service, appErr := buildService(d.ID, d.Manifest)
	if appErr != nil {
		logger.Error(errors.Wrap(appErr, "failed to load service, this is most likely a bug or a service schema change issue"))
		return model.ErrUnknown
	}

	env := s.buildEnvironment(project)
	if err := env.Validate(); err != nil {
		logger.Error(errors.Wrap(appErr, "failed to validate the environment , this is most likely a bug or a project schema change issue"))
		return model.ErrUnknown
	}

	env.Provider.LoadDefaultCluster()

	injectSecrets(service, project.LoadedSettings.Secrets)

	l, reader := getLogger()

	// done is used by saveLogs to know when the the deployment is done
	// wait is used by the this function to know when saveLogs is done writing to the DB
	done := make(chan bool, 1)
	wait := &sync.WaitGroup{}
	wait.Add(1)

	go s.saveLogs(activityID, reader, done, wait)
	err := f(service, env, l)

	done <- true
	wait.Wait()

	return err
}

func (s *Server) buildEnvironment(project *model.Project) *model.Environment {
	e := &model.Environment{}
	e.Name = project.DNSName
	e.ProjectName = project.Name
	e.ID = project.ID
	e.Provider = project.LoadedSettings.Provider
	e.Registry = project.LoadedSettings.Registry
	e.DNSProvider = s.DNSProvider
	return e
}

func buildService(serviceID, manifest string) (*model.Service, *model.AppError) {
	s, err := model.ParseEncodedManifest(manifest)
	if err != nil {
		return nil, err
	}

	s.ID = serviceID

	return s, nil
}

func orDefault(v string, d string) string {
	if v == "" {
		return d
	}

	return v
}

func getLogger() (*log.Logger, *bufio.Reader) {
	w := new(bytes.Buffer)
	l := &log.Logger{}
	l.SetOutput(w)
	r := bufio.NewReader(w)
	return l, r
}

func (s *Server) saveLogs(activityID string, reader *bufio.Reader, done <-chan bool, wait *sync.WaitGroup) {
	s.pendingOperations.Add(1)
	defer s.pendingOperations.Done()

	defer wait.Done()
	isDone := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("failed to read from the log stream for activity-%s: %s", activityID, err.Error())
			}

			if isDone {
				return
			}

			// We get an EOF when the buffer doesn't have any more logs. We use the done channel to know if the deployment activity
			// has finished, or if we should expect more logs. If the deployment is done, we know that the next EOF is the true end
			// of the logger.
			select {
			case <-done:
				isDone = true
			default:
				time.Sleep(5 * time.Millisecond)
				continue
			}
		}

		line = strings.TrimSpace(line)
		if line != "" {
			s.addLog(activityID, line)
		}
	}
}

func injectSecrets(service *model.Service, secrets []*model.EnvVar) {
	for c := range service.Containers {
		for _, e := range service.Containers[c].Environment {
			if e.Value == "" {
				for _, s := range secrets {
					if e.Name == s.Name {
						e.Value = s.Value
					}
				}
			}

			if strings.HasPrefix(e.Value, "$") {
				secretName := e.Value[1:]
				e.Value = ""
				for _, s := range secrets {
					if secretName == s.Name {
						e.Value = s.Value
					}
				}
			}
		}
	}
}

func (s *Server) waitForProvider() {

	c := make(chan struct{})
	go func() {
		defer close(c)
		s.pendingOperations.Wait()
	}()
	select {
	case <-c:
		log.Println("provider transactions completed")
		return // completed normally
	case <-time.After(providerTimeout):
		log.Printf("provider transactions were not completed after %f seconds", providerTimeout.Seconds())
		return // timed out
	}
}
