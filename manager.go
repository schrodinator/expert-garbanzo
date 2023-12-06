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
	m.handlers[EventEnterRoom]   = ChatRoomHandler
	m.handlers[EventChangeTeam]  = TeamChangeHandler
	m.handlers[EventChangeRole]  = RoleChangeHandler
	m.handlers[EventNewGame]     = NewGameHandler
	m.handlers[EventMakeGuess]   = GuessEvaluationHandler
	m.handlers[EventGiveClue]    = ClueHandler
	m.handlers[EventAbortGame]   = AbortGameHandler
	m.handlers[EventEndTurn]     = EndTurnHandler
}

func NewGameHandler(event Event, c *Client) error {
	m := c.manager

	var gameRequest NewGameRequestEvent
	if err := json.Unmarshal(event.Payload, &gameRequest); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	/* All clients in the chat room at the time of
	   game creation are added as players. */
	game := m.makeGame(c.chatroom, m.chats[c.chatroom], &gameRequest.Bots)

	/* Ensure initial game state is valid. */
	if !game.validGame() {
		game.notifyPlayers(EventGameOver, nil)
		return fmt.Errorf("Invalid game state requested")
	}

	cluegiverMessage := NewGameResponseEvent {
		Cards: game.cards,
		SentTime: time.Now(),
	}
	cluegiverEvent, err := packageMessage(EventNewGame, cluegiverMessage)
	if err != nil {
		return err
	}

	guesserMessage := NewGameResponseEvent {
		Cards: game.cards.whiteCards(),
		SentTime: cluegiverMessage.SentTime,
	}
	guesserEvent, err := packageMessage(EventNewGame, guesserMessage)
	if err != nil {
		return err
	}

	/* Send appropriately colored cards based on role. */
	for _, player := range game.players {
		if player.role == cluegiver {
			player.egress <- cluegiverEvent
		} else {
			player.egress <- guesserEvent
		}
	}

	return game.botPlay(GiveClueEvent{})
}

func AbortGameHandler(event Event, c *Client) error {
	c.game = nil
	game, exists := c.manager.games[c.chatroom]
	if !exists {
		return fmt.Errorf("Game %v not found", c.chatroom)
	}
	delete(game.players, c.username)
	if game.removeGame() {
		return nil
	}

	game.actions[c.team][c.role] -= 1

	abortGame := AbortGameEvent {
		UserName: c.username,
		TeamColor: c.team,
	}
	return game.notifyPlayers(EventAbortGame, abortGame)
}

func EndTurnHandler(event Event, c *Client) error {
	game := c.game
	game.changeTurn()

	payload := EndTurnEvent {
		TeamTurn: game.teamTurn,
		RoleTurn: game.roleTurn,
	}
	if err := game.notifyPlayers(EventEndTurn, payload); err != nil {
		return err
	}
	return game.botPlay(GiveClueEvent{})
}

func GuessEvaluationHandler(event Event, c *Client) error {
	game := c.game
	if c.team != game.teamTurn {
		return errors.New("It is not this player's team turn")
	}
	if c.role != game.roleTurn {
		return fmt.Errorf(
			"It is not this player's role turn. Player role: %v, game role: %v",
			c.role, game.roleTurn)
	}

	var guessResponse GuessResponseEvent
	if err := json.Unmarshal(event.Payload, &guessResponse); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}
	card := guessResponse.Guess
	cardColor := game.cards[guessResponse.Guess]
	guessResponse.Correct = game.evaluateGuess(cardColor)
	guessResponse.GuessRemaining = game.guessRemaining
	guessResponse.TeamColor = c.team
	guessResponse.CardColor = cardColor
	guessResponse.TeamTurn  = game.teamTurn
	guessResponse.RoleTurn  = game.roleTurn
	guessResponse.Score     = game.score

	game.cards[card] = "guessed-" + cardColor

	if err := game.notifyPlayers(EventMakeGuess, guessResponse); err != nil {
		return err
	}
	if cardColor == deathCard || game.score[c.team] <= 0 {
		game.removeGame()
		return nil
	}
	if guessResponse.Correct && game.guessRemaining > 0 {
		/* It's still the current guesser's turn.
		   Short circuit the call to bots. */
		return nil
	}
	return game.botPlay(GiveClueEvent{})
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

	err := game.notifyPlayers(EventGiveClue, event.Payload)
	if err != nil {
		return err
	}
	return game.botPlay(clue)
}

func ChatRoomHandler(event Event, c *Client) error {
	var changeroom ChangeRoomEvent

	if err := json.Unmarshal(event.Payload, &changeroom); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	oldroom := c.chatroom
	newroom := changeroom.RoomName

	// if client is in newroom already, there is nothing to do
	if oldroom == newroom {
		return nil
	}

	// remove client from old chat room
	delete(c.manager.chats[oldroom], c.username)

	// notify old chat room that client has left
	c.manager.notifyClients(oldroom, EventExitRoom, event.Payload)

	// notify new room that the client is entering
	c.manager.notifyClients(newroom, EventEnterRoom, event.Payload)

	// enter client into new chat room
	c.chatroom = newroom
	c.manager.makeChatRoom(newroom)
	c.manager.chats[newroom][c.username] = c

	// notify client if there is a game in progress
	if _, exists:= c.manager.games[newroom]; exists {
		if len(c.manager.games[newroom].players) > 0 {
			changeroom.GameInProgress = true
		}
	}

	// send list of current chat room participants to client
	changeroom.Participants = c.manager.chats[newroom].listClients()
	outgoingEvent, err := packageMessage(EventEnterRoom, changeroom)
	c.egress <- outgoingEvent
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

	broadMessage := NewMessageEvent {
		SentTime: time.Now(),
		SendMessageEvent: SendMessageEvent {
			Message: chatevent.Message,
			From: chatevent.From,
			Color: chatevent.Color,
		},
	}

	status := c.manager.notifyClients(c.chatroom, EventNewMessage, broadMessage)
	return status
}

func (m *Manager) notifyClients(room string, messageType string, message any) error {
	if _, exists := m.chats[room]; !exists {
		return fmt.Errorf("chat room %v does not exist", room)
	}

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

func (m *Manager) makeGame(name string, players ClientList, bots *BotActions) *Game {
	game, exists := m.games[name]
	if exists {
		return game
	}
	game = &Game {
		name: name,
		players: players,
		cards: getCards(),
		actions: getActions(players, bots),
		teamTurn: red,
		roleTurn: cluegiver,
		score: Score {
			red: 9,
			blue: 8,
		},
		manager: m,
	}
	game.makeBot(bots)
	m.games[name] = game
	
	for _, player := range players {
		player.game = game
	}

	return game
}

/* Return true if game was deleted. */
func (m *Manager) removeGame(room string) bool {
	game := m.games[room]
	if game != nil {
		if len(game.players) == 0 {
			delete(m.games, room)
			return true
		}
		m.notifyClients(room, EventGameOver, nil)
	}
	return false
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
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	room := client.chatroom
	if _, exists := m.games[room]; exists {
		delete(m.games[room].players, client.username)
		// remove the game if no more players
		m.removeGame(room)
	}
	if _, exists := m.chats[room]; exists {
		delete(m.chats[room], client.username)
	}
	if _, exists := m.clients[client.username]; exists {
		client.connection.Close()
		// notify chat room of client departure
		exit := ChangeRoomEvent {
			UserName: client.username,
		}
		m.notifyClients(room, EventExitRoom, exit)
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
