package events

import (
	"log"

	"bitbucket.org/okteto/okteto/backend/model"
)

//Auth is the function used to authenticate a token based on the information received
type Auth func(string) (*model.User, error)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	messages chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

// NewHub initializes the channels and maps in newhub
func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts accepting connections and disconnections
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

//SendNotification sends a notification to all the clients subscribed to the project
func (h *Hub) SendNotification(s *model.Service) {
	go func() {
		m := &Message{MessageType: serviceMessage, Service: s}
		for k := range h.clients {
			if !k.authenticated {
				log.Printf("ws-%s not yet authenticated", k.id)
				continue
			}

			if k.project == s.ProjectID {
				select {
				case k.send <- m:
				default:
					close(k.send)
					delete(h.clients, k)
				}
			}
		}
	}()
}
