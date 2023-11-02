package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	clients  ClientList
	games    GameMap
	handlers EventHandlerList

	sync.RWMutex

	otps RetentionMap
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients:  make(ClientList),
		games:    make(GameMap),
		handlers: make(map[string]EventHandler),
		otps:     NewRetentionMap(ctx, 5*time.Second),
	}
	m.setupEventHandlers()

	// TODO: May prefer to disable "New Game" button in default chatroom
	// and remove the lines below. Doing so would treat the default chatroom
	// as a lobby; players must switch to a new chatroom to play games.
	// Otherwise, creating this empty Game object at the outset is necessary
	// to give new clients a place to be appended to. This is otherwise
	// handled in ChatRoomHandler when changing rooms.
	m.games[defaultChatroom] = Game{
		players: make(ClientList),
	}

	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventSendMessage] = SendMessage
	m.handlers[EventChangeRoom]  = ChatRoomHandler
	m.handlers[EventChangeTeam]  = TeamChangeHandler
	m.handlers[EventChangeRole]  = RoleChangeHandler
	m.handlers[EventNewGame]     = NewGameHandler
	m.handlers[EventMakeGuess]   = GuessEvaluationHandler
	m.handlers[EventGiveClue]    = ClueHandler
}

func NewGameHandler(event Event, c *Client) error {
	cards := getGameWords()

	var guesserMessage NewGameEvent
	guesserMessage.Words = cards
	guesserMessage.Sent  = time.Now()

	guesserData, err := json.Marshal(guesserMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal guesser message: %v", err)
	}

	getAlignments(cards)

	var cluegiverMessage NewGameEvent
	cluegiverMessage.Words = cards
	cluegiverMessage.Sent  = guesserMessage.Sent

	cluegiverData, err := json.Marshal(cluegiverMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal cluegiver message: %v", err)
	}

	guesserEvent := Event {
		Type:    EventNewGame,
		Payload: guesserData,
	}

	cluegiverEvent := Event {
		Type:    EventNewGame,
		Payload: cluegiverData,
	}

	game := c.manager.games[c.chatroom]
	game.cards = cards
	c.manager.games[c.chatroom] = game

	for _, client := range game.players {
		if client.role == "cluegiver" {
			client.egress <- cluegiverEvent
		} else {
			client.egress <- guesserEvent
		}
	}
	return nil	
}

func GuessEvaluationHandler(event Event, c *Client) error {
	var guessResponse GuessResponseEvent

	if err := json.Unmarshal(event.Payload, &guessResponse); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	game := c.manager.games[c.chatroom]
	alignment := game.cards[guessResponse.Guess]

	guessResponse.GuesserTeam = c.team;
	guessResponse.CardAlignment = alignment;
	if (c.team == alignment) {
		guessResponse.Correct = true;
	}

	data, err := json.Marshal(guessResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %v", err)
	}

	outgoingEvent := Event {
		Type:    EventMakeGuess,
		Payload: data,
	}

	for _, client := range game.players {
		client.egress <- outgoingEvent
	}

	return nil
}

func ClueHandler (event Event, c *Client) error {
	game := c.manager.games[c.chatroom]
	for _, client := range game.players {
		client.egress <- event
	}
	return nil
}

func ChatRoomHandler(event Event, c *Client) error {
	var changeroom ChangeRoomEvent

	if err := json.Unmarshal(event.Payload, &changeroom); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	room := changeroom.RoomName
	c.chatroom = room
	game, exists := c.manager.games[room]
	if !exists {
		game = Game {
			players: make(ClientList),
		}
	}
	game.players[c.username] = c
	c.manager.games[room] = game

	// Report to everyone in the room that the new player has entered
	for _, client := range game.players {
		client.egress <- event
	}

	return nil
}

func TeamChangeHandler(event Event, c *Client) error {
	if c.team == defaultTeam {
		c.team = otherTeam
	} else {
		c.team = defaultTeam
	}
	return nil
}

func RoleChangeHandler(event Event, c *Client) error {
	if c.role == defaultRole {
		c.role = otherRole
	} else {
		c.role = defaultRole
	}
	return nil
}

func SendMessage(event Event, c *Client) error {
	var chatevent SendMessageEvent

	if err := json.Unmarshal(event.Payload, &chatevent); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	var broadMessage NewMessageEvent

	broadMessage.Sent = time.Now()
	broadMessage.Message = chatevent.Message
	broadMessage.From = chatevent.From
	broadMessage.Color = chatevent.Color

	data, err := json.Marshal(broadMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %v", err)
	}

	outgoingEvent := Event{
		Type:    EventNewMessage,
		Payload: data,
	}

	for _, client := range c.manager.games[c.chatroom].players {
		client.egress <- outgoingEvent
	}

	return nil
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("there is no such event type")
	}
}

func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	username := m.otps.VerifyOTP(otp)
	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Println("new connection")

	// upgrade regular http connection into websocket
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(username, conn, m)

	m.addClient(client)

	// Start client processes
	go client.readMessages()
	go client.writeMessages()
}

func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type response struct {
		OTP     string `json:"otp"`
		Message string `json:"message"`
	}

	var (
		req  userLoginRequest
		resp response
	)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// replace with real authentication
	if req.Password != masterPassword {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Enforce unique usernames
	if _, exists := m.clients[req.Username]; exists {
		// someone with this username is already logged in
		resp.Message = "Username " + req.Username + " is already logged in. Choose a different username."
	} else {
		otp := m.otps.NewOTP(req.Username)
		resp.OTP = otp.Key
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Println(err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client.username] = client
	// TODO: May prefer to disable "New Game" button in default chatroom
	// and remove the line below. This would treat the default chatroom as a
	// lobby; players would have to switch to a new chatroom to play games.
	m.games[defaultChatroom].players[client.username] = client
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, exists := m.clients[client.username]; exists {
		client.connection.Close()
		delete(m.clients, client.username)
	}
	if _, exists := m.games[client.chatroom]; exists {
		delete(m.games[client.chatroom].players, client.username)
	}
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "https://localhost:8080":
		return true
	default:
		return false
	}
}
