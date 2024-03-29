package main

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = 5 * time.Second
)

type Participant struct {
	Name   string  `json:"name"`
	Team   Team    `json:"teamColor"`
	Role   Role    `json:"role"`
	InGame bool    `json:"inGame"`
}

type ClientList map[string]*Client
/* Alphabetical list of clients. */
func (cl ClientList) listClients() []Participant {
	participants := make([]Participant, len(cl))
	i := 0
	for name, client := range cl {
		inGame := false
		if client.game != nil {
			inGame = true
		}
		participants[i] = Participant {
			Name: name,
			Team: client.team,
			Role: client.role,
			InGame: inGame,
		}
		i++
	}
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Name < participants[j].Name
	})
	return participants
}

type ChatRooms map[string]ClientList

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	game       *Game
	username   string
	chatroom   string
	team       Team
	role       Role

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}

func NewClient(username string, conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		username:   username,
		role:       guesser,
		team:       red,
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
		log.Error().Err(err)
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
			log.Error().Err(err).Msg("error unmarshalling event")
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Error().Str("event", request.Type).Err(err)
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Error().Err(err).Msg("error closing websocket")
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Error().Err(err)
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Error().Err(err).Msg("failed to send TextMessage over websocket")
			}

		// Heartbeats
		case <-ticker.C:
			// Send a Ping to the Client
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Error().Err(err).Msg("failed to send PingMessage over websocket")
				return
			}
		}
	}
}

// Heartbeats - reset the timer
func (c *Client) pongHandler(pongMsg string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
