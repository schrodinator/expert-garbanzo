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
	chats    ChatRooms
	games    GameList
	handlers EventHandlerList
	bot      *Bot

	sync.RWMutex

	otps RetentionMap
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients:  make(ClientList),
		chats:    make(ChatRooms),
		games:    make(GameList),
		handlers: make(map[string]EventHandler),
		otps:     NewRetentionMap(ctx, 5*time.Second),
	}

	m.setupEventHandlers()

	m.makeChatRoom(defaultChatRoom)

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
	m.handlers[EventEndTurn]     = EndTurnHandler
}

func NewGameHandler(event Event, c *Client) error {
	game := c.manager.makeGame(c.chatroom)

	var cluegiverMessage NewGameEvent
	cluegiverMessage.Cards = game.cards
	cluegiverMessage.SentTime = time.Now()
	cluegiverEvent, err := packageMessage(EventNewGame, cluegiverMessage)
	if err != nil {
		return err
	}

	var guesserMessage NewGameEvent
	guesserMessage.Cards = whiteCards(game.cards)
	guesserMessage.SentTime  = cluegiverMessage.SentTime
	guesserEvent, err := packageMessage(EventNewGame, guesserMessage)
	if err != nil {
		return err
	}

	for _, client := range c.manager.chats[c.chatroom] {
		/* All clients in the chat room at the time of
	       game creation are added as players */
		game.players[client.username] = client
		if _, exists := game.teamCounts[client.team]; exists {
			game.teamCounts[client.team] += 1
		} else {
			game.teamCounts[client.team] = 1
		}
		client.game = game

		if client.role == cluegiver {
			client.egress <- cluegiverEvent
		} else {
			client.egress <- guesserEvent
		}
	}

	return nil	
}

func AbortGameHandler(event Event, c *Client) error {
	c.game = nil
	game, exists := c.manager.games[c.chatroom]
	if !exists {
		return fmt.Errorf("Game %v not found", c.chatroom)
	}
	delete(game.players, c.username)

	game.teamCounts[c.team] -= 1
	if (game.teamCounts[c.team] <= 0) {
		delete(game.teamCounts, c.team)
	}

	var abortGame AbortGameEvent
	abortGame.UserName = c.username
	abortGame.TeamColor = c.team

	err := c.manager.notifyPlayers(c.chatroom, EventAbortGame, abortGame)
	return err
}

func EndTurnHandler(event Event, c *Client) error {
	game := c.game
	game.changeTurn()

	var payload EndTurnEvent
	payload.TeamTurn = game.teamTurn
	payload.RoleTurn = game.roleTurn
	c.manager.notifyPlayers(c.chatroom, "end_turn", payload)

	return nil
}

func GuessEvaluationHandler(event Event, c *Client) error {
	var guessResponse GuessResponseEvent

	if err := json.Unmarshal(event.Payload, &guessResponse); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	game := c.game
	card := guessResponse.Guess
	cardColor := game.cards[guessResponse.Guess]
	if c.team != game.teamTurn {
		return errors.New("It is not this player's team turn")
	}
	if c.role != game.roleTurn {
		return fmt.Errorf(
			"It is not this player's role turn. Player role: %v, game role: %v",
			c.role, game.roleTurn)
	}

	guessResponse.Correct = game.evaluateGuess(cardColor)
	guessResponse.GuessRemaining = game.guessRemaining
	guessResponse.TeamColor = c.team
	guessResponse.CardColor = cardColor
	guessResponse.TeamTurn  = game.teamTurn
	guessResponse.RoleTurn  = game.roleTurn
	guessResponse.Score     = game.score

	game.cards[card] = "guessed-" + cardColor

	err := c.manager.notifyPlayers(c.chatroom, EventMakeGuess, guessResponse)
	return err
}

func ClueHandler(event Event, c *Client) error {
	game := c.game

	// if we're here, a clue was given; now it's the guesser's turn
	game.roleTurn = guesser

	var clue GiveClueEvent
	if err := json.Unmarshal(event.Payload, &clue); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}
	if (clue.NumCards == 0) {
		/* Special case: if the cluegiver did not specify the number
		   of cards, their team gets unlimited guesses. Set the
		   number of guesses equal to the number of cards in the game. */
		game.guessRemaining = totalNumCards
	} else {
		game.guessRemaining = clue.NumCards + 1
	}

	err := c.manager.notifyPlayers(c.chatroom, EventGiveClue, event.Payload)
	return err
}

func ChatRoomHandler(event Event, c *Client) error {
	var changeroom ChangeRoomEvent

	if err := json.Unmarshal(event.Payload, &changeroom); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	// remove client from old chat room
	delete(c.manager.chats[c.chatroom], c.username)

	// enter client into new chat room
	c.chatroom = changeroom.RoomName
	c.manager.makeChatRoom(c.chatroom)
	c.manager.chats[c.chatroom][c.username] = c

	// Report to everyone in the room that the new player has entered
	err := c.manager.notifyClients(c.chatroom, EventChangeRoom, event.Payload)
	return err
}

func TeamChangeHandler(event Event, c *Client) error {
	c.team = c.team.Change()
	return nil
}

func RoleChangeHandler(event Event, c *Client) error {
	c.role = c.role.Change()
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

	status := c.manager.notifyClients(c.chatroom, EventNewMessage, broadMessage)
	return status
}

func (m *Manager) notifyPlayers(room string, messageType string, message any) error {
	outgoingEvent, err := packageMessage(messageType, message)
	if err != nil {
		return err
	}

	for _, client := range m.games[room].players {
		client.egress <- outgoingEvent
	}

	return nil
}

func (m *Manager) notifyClients(room string, messageType string, message any) error {
	outgoingEvent, err := packageMessage(messageType, message)
	if err != nil {
		return err
	}

	for _, client := range m.chats[room] {
		client.egress <- outgoingEvent
	}

	return nil
}

func packageMessage(messageType string, message any) (Event, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return Event{}, fmt.Errorf(
			"failed to marshal broadcast message of type %v: %v", messageType, err)
	}

	return Event {
		Type:    messageType,
		Payload: data,
	}, nil
}

func (m *Manager) makeChatRoom(name string) {
	if _, exists := m.chats[name]; exists {
		return
	}
	m.chats[name] = make(ClientList)
}

func (m *Manager) makeGame(name string) *Game {
	game, exists := m.games[name]
	if !exists {
		game = &Game {
			players: make(ClientList),
			teamCounts: make(map[Team]int),
		}
	}
	game.cards = getCards()
	game.teamTurn = red
	game.roleTurn = cluegiver
	game.score = make(Score)
	game.score[red] = 9
	game.score[blue] = 8
	m.games[name] = game
	return game
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
	m.chats[defaultChatRoom][client.username] = client
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	// TODO: delete empty games / chatrooms
	if _, exists := m.games[client.chatroom]; exists {
		delete(m.games[client.chatroom].players, client.username)
	}
	if _, exists := m.chats[client.chatroom]; exists {
		delete(m.chats[client.chatroom], client.username)
	}
	if _, exists := m.clients[client.username]; exists {
		client.connection.Close()
		delete(m.clients, client.username)
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
