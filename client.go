package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = pongWait * 9 / 10
)

type ClientList map[string]*Client

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	username   string
	chatroom   string
	role       string

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}

func NewClient(username string, conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		username:   username,
		chatroom:   defaultChatroom,
		role:       defaultRole,
		egress:     make(chan Event),
	}
}

func (c *Client) readMessages() {
	defer func() {
		// cleanup connection
		c.manager.removeClient(c)
	}()

	// Heartbeats
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}
	c.connection.SetPongHandler(c.pongHandler)

	// Fix for jumbo frame (don't let people overflow buffer)
	// This will close connection with the offending client
	c.connection.SetReadLimit(512)

	for {
		_, payload, err := c.connection.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		var request Event

		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("error unmarshalling event: %v", err)
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Println(err)
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("connection closed: ", err)
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("failed to send message: ", err)
			}
			log.Println("message sent")

		// Heartbeats
		case <-ticker.C:
			log.Println("ping")

			// Send a Ping to the Client
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("writemsg err: ", err)
				return
			}
		}
	}
}

// Heartbeats - reset the timer
func (c *Client) pongHandler(pongMsg string) error {
	log.Println("pong")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
