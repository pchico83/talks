package app

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"

	"golang.org/x/net/context"

	"bitbucket.org/okteto/okteto/backend/events"
)

//Server holds the dependencies for the API server
type Server struct {
	API               *http.Server
	Metrics           *http.Server
	Hub               *events.Hub
	Email             *EmailProvider
	DNSProvider       *model.DNSProvider
	Listener          *pq.Listener
	pendingOperations sync.WaitGroup
	DB                *gorm.DB
}

//Start starts the http server on the specified listening address
func (s *Server) Start() {
	go func() {

		go s.Hub.Run()
		go s.processEvents()
		go s.processGithubEvents()
		go s.sync()
		go s.cleanExpiredServices()

		go func(h *http.Server) {
			err := h.ListenAndServe()
			if err != nil {
				logger.Info("metric server exiting: %s", err.Error())
			}
		}(s.Metrics)

		var err error
		err = s.API.ListenAndServe()
		if err != nil {
			if err != http.ErrServerClosed {
				logger.Error(errors.Wrap(err, "http server closed"))
			}
		}
	}()
}

//Stop stops the http server. It sends a shutdown signal that gives the actin requests 5 seconds
// to finish before they get shutdown.
func (s *Server) Stop() {
	log.Printf("stopping http server")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	s.API.Shutdown(ctx)
	log.Println("waiting for pending okteto transactions to exit")
	s.waitForProvider()
}
