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
	m.handlers[EventAbortGame]   = AbortGameHandler
}

func NewGameHandler(event Event, c *Client) error {
	cards := getGameWords()

	var guesserMessage NewGameEvent
	guesserMessage.Cards = cards
	guesserMessage.SentTime  = time.Now()

	guesserData, err := json.Marshal(guesserMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal guesser message: %v", err)
	}

	getCardColors(cards)

	var cluegiverMessage NewGameEvent
	cluegiverMessage.Cards = cards
	cluegiverMessage.SentTime  = guesserMessage.SentTime

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
	game.teamTurn = redTeam
	game.roleTurn = cluegiverRole
	c.manager.games[c.chatroom] = game

	for _, client := range game.players {
		if client.role == cluegiverRole {
			client.egress <- cluegiverEvent
		} else {
			client.egress <- guesserEvent
		}
	}
	return nil	
}

func AbortGameHandler(event Event, c *Client) error {
	m := c.manager
	if _, exists := m.games[c.chatroom]; exists {
		delete(m.games[c.chatroom].players, c.username)
	}

	var abortGame AbortGameEvent
	abortGame.UserName = c.username
	abortGame.TeamColor = c.team

	data, err := json.Marshal(abortGame)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %v", err)
	}

	outgoingEvent := Event {
		Type:    EventAbortGame,
		Payload: data,
	}

	for _, client := range m.games[c.chatroom].players {
		client.egress <- outgoingEvent
	}

	return nil
}

func GuessEvaluationHandler(event Event, c *Client) error {
	var guessResponse GuessResponseEvent

	if err := json.Unmarshal(event.Payload, &guessResponse); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	game := c.manager.games[c.chatroom]
	card := guessResponse.Guess
	cardColor := game.cards[guessResponse.Guess]
	if (c.team != game.teamTurn) {
		return errors.New("It is not this player's team turn")
	}
	if (c.role != game.roleTurn) {
		return fmt.Errorf("It is not this player's role turn. Player role: %v, game role: %v", c.role, game.roleTurn)
	}

	guessResponse.GuesserTeam = c.team
	guessResponse.CardColor = cardColor
	if (c.team == cardColor) {
		// Correct guess. Guessers on this team may continue guessing.
		guessResponse.Correct = true
		guessResponse.TeamTurn = c.team
		guessResponse.RoleTurn = guesserRole
	} else if (cardColor != deathCard) {
		if (c.team == redTeam) {
			guessResponse.TeamTurn = blueTeam
			game.teamTurn = blueTeam
		} else {
			guessResponse.TeamTurn = redTeam
			game.teamTurn = redTeam
		}
		// Cluegiver always starts
		guessResponse.RoleTurn = cluegiverRole
		game.roleTurn = cluegiverRole
	}

	game.cards[card] = "guessed-" + cardColor
	c.manager.games[c.chatroom] = game

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

	// if we're here, a clue was given; now it's the guesser's turn
	game.roleTurn = guesserRole
	c.manager.games[c.chatroom] = game

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
	if c.team == redTeam {
		c.team = blueTeam
	} else {
		c.team = redTeam
	}
	return nil
}

func RoleChangeHandler(event Event, c *Client) error {
	if c.role == guesserRole {
		c.role = cluegiverRole
	} else {
		c.role = guesserRole
	}
	return nil
}

func SendMessage(event Event, c *Client) error {
	var chatevent SendMessageEvent

	if err := json.Unmarshal(event.Payload, &chatevent); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	var broadMessage NewMessageEvent

	broadMessage.SentTime = time.Now()
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
