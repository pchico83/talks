package app

import (
	"encoding/json"
	"log"
	"time"

	"bitbucket.org/okteto/okteto/backend/model"
)

//Notification represents the notifications raised from the DB
type Notification struct {
	Table   string
	Action  string
	ID      string
	Project string
}

func (s *Server) processEvents() {
	for {
		select {
		case n := <-s.Listener.Notify:
			var notification Notification
			err := json.Unmarshal([]byte(n.Extra), &notification)
			if err != nil {
				log.Println("Error processing JSON: ", err)
				continue
			}

			if notification.Table == "services" {
				s.fireHubNotification(notification.Project, notification.ID)
			}

		case <-time.After(90 * time.Second):
			go func() {
				err := s.Listener.Ping()
				if err != nil {
					log.Printf("ERROR: can't reach out the DB: %s", err.Error())
				}
			}()
		}
	}
}

func (s *Server) fireHubNotification(projectID string, serviceID string) {
	var project model.Project
	s.DB.Where("id = ?", projectID).First(&project)
	d, err := s.GetServiceAndActivities(&project, serviceID)
	if err != nil {
		log.Printf("error when getting service for notifications: %+v", err)
		return
	}

	s.Hub.SendNotification(d)
}
