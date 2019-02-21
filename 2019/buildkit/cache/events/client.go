package events

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	authTimeout = (30 * time.Second)
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

const (
	authMessage    = MessageType("auth")
	serviceMessage = MessageType("service")
)

// MessageType is the type of event send or received
type MessageType string

// Message sent and received via the socket
type Message struct {
	MessageType MessageType    `json:"type,omitempty"`
	Token       string         `json:"token,omitempty"`
	Service     *model.Service `json:"service,omitempty"`
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	id string

	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan *Message

	// The project the client is interested on
	project string

	// The user
	user string

	// Is it already authenticated?
	authenticated bool

	authFn Auth

	// The connection start time
	started time.Time

	ticker *time.Ticker
}

func (c *Client) stop(reason string) {
	c.ticker.Stop()
	c.conn.Close()
	c.hub.unregister <- c
	logger.Info(fmt.Sprintf("ws-%s closed: %s", c.id, reason))
}

// writePump pumps messages to the websocket
func (c *Client) writePump() {
	c.ticker = time.NewTicker(pingPeriod)
	defer c.stop("writePump exited")
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().UTC().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				logger.Info("ws-%s error when setting the write deadline", c.id)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				logger.Error(errors.Wrapf(err, "ws-%s error when writing a message", c.id))
				return
			}

		case <-c.ticker.C:
			if !c.authenticated && time.Now().Sub(c.started) > authTimeout {
				log.Printf("ws-%s not authenticated, disconnecting", c.id)
				return
			}

			c.conn.SetWriteDeadline(time.Now().UTC().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error(errors.Wrapf(err, "ws-%s error when writing ping message", c.id))
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer c.stop("readPump exited")
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		m := &Message{}
		err := c.conn.ReadJSON(m)
		if err != nil {
			if _, k := err.(*websocket.CloseError); k {
				return
			}

			logger.Error(errors.Wrapf(err, "ws-%s fail to read message, closing it", c.id))
			return
		}

		if m.MessageType == authMessage {
			if m.Token == "" {
				logger.Error(fmt.Errorf("ws-%s is missing the auth token", c.id))
				continue
			}

			c.finishAuth(m.Token)
		} else {
			logger.Error(fmt.Errorf("ws-%s received an unknown event: %s", c.id, m.MessageType))
		}
	}
}

func (c *Client) startAuth() {
	m := &Message{MessageType: authMessage}
	c.send <- m
}

func (c *Client) finishAuth(token string) {
	u, err := c.authFn(token)
	if err != nil {
		logger.Error(errors.Wrapf(err, "ws-%s failed authentication, closing", c.id))
		c.stop("auth failed")
		return
	}

	c.authenticated = true
	c.user = u.ID
	log.Printf("ws-%s authenticated successfully with user-%s", c.id, u.ID)
}

//StartNewClient registers and starts processing requests from a new client
func (hub *Hub) StartNewClient(conn *websocket.Conn, projectID string, authFn Auth) string {
	client := &Client{
		id:            uuid.NewV4().String(),
		hub:           hub,
		conn:          conn,
		send:          make(chan *Message, 256),
		project:       projectID,
		authenticated: false,
		authFn:        authFn,
		started:       time.Now(),
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.startAuth()
	go client.readPump()
	return client.id
}
