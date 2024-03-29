package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	clients  ClientList
	chats    ChatRooms
	games    GameList
	handlers EventHandlerList

	sync.RWMutex

	otps RetentionMap
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients:  make(ClientList),
		chats:    make(ChatRooms),
		games:    make(GameList),
		handlers: make(EventHandlerList),
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
	game, err := m.makeGame(c.chatroom, m.chats[c.chatroom], &gameRequest.Bots)
	/* Ensure game was created (valid initial state). */
	if err != nil {
		m.notifyClients(c.chatroom, EventInvalidState,
			"Need one guesser and one cluegiver per team.")
		return fmt.Errorf("invalid game state requested")
	}

	cluegiverMessage := NewGameResponseEvent {
		Cards: game.cards,
		TeamTurn: game.teamTurn,
	}
	cluegiverEvent, err := packageMessage(EventNewGame, cluegiverMessage)
	if err != nil {
		return err
	}

	guesserMessage := NewGameResponseEvent {
		Cards: game.cards.whiteCards(),
		TeamTurn: game.teamTurn,
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
	game, exists := c.manager.games[c.chatroom]
	if !exists {
		return fmt.Errorf("Game %v not found", c.chatroom)
	}

	abortGame := PlayerAlignmentResponse {
		UserName: c.username,
		TeamColor: c.team,
		Role: c.role,
	}
	if err := c.manager.notifyClients(c.chatroom, EventAbortGame, abortGame); err != nil {
		return err
	}

	game.removePlayer(c.username)
	if len(game.players) == 0 {
		if !game.removeGame() {
			return fmt.Errorf("could not remove game %v", c.chatroom)
		}
		return nil
	}

	if game.active {
		if !game.validGame() {
			/* TODO: consider having a bot fill in for any unfilled role
			as long as there is at least one remaining human player. */
			game.notifyPlayers(EventInvalidState, "Essential roles unfilled. Cannot continue the game.")
			game.removeGame()
		}
	}
	return nil
}

func EndTurnHandler(event Event, c *Client) error {
	game := c.game
	if game == nil {
		return fmt.Errorf("game does not exist")
	}
	if !game.active {
		return fmt.Errorf("inactive game")
	}
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
	if game == nil {
		return fmt.Errorf("game does not exist")
	}
	if !game.active {
		return fmt.Errorf("inactive game")
	}
	if c.team != game.teamTurn {
		return errors.New("player team doesn't match team turn")
	}
	if c.role != game.roleTurn {
		return fmt.Errorf("player role doesn't match role turn")
	}

	var guessResponse GuessResponseEvent
	if err := json.Unmarshal(event.Payload, &guessResponse); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}
	guessResponse.TeamColor = c.team
	if GuessEvaluation(guessResponse, c) {
		/* It's still the current guesser's turn. */
		return nil
	}
	return game.botPlay(GiveClueEvent{})
}

func GuessEvaluation(guessResponse GuessResponseEvent, c *Client) bool {
	game := c.game
	guess := guessResponse.Guess
	if _, exists := game.cards[guess]; !exists {
		return false
	}
	cardColor := game.cards[guess]
	guessResponse.Correct = game.evaluateGuess(cardColor)
	guessResponse.GuessRemaining = game.guessRemaining
	guessResponse.CardColor = cardColor
	guessResponse.TeamTurn  = game.teamTurn
	guessResponse.RoleTurn  = game.roleTurn
	guessResponse.Score     = game.score

	game.cards[guess] = "guessed-" + cardColor
	game.notifyPlayers(EventMakeGuess, guessResponse)

	if cardColor == "neutral" {
		return false
	}

	if cardColor == deathCard {
		t := c.team.Title()
		game.removeGame(fmt.Sprintf("%v Team uncovers the Black Card. %v Team loses!", t, t))
		return false
	}

	t := Team(cardColor)
	if game.score[t] <= 0  {
		game.removeGame(fmt.Sprintf("%v Team wins!", t.Title()))
		return false
	}
	
	if guessResponse.Correct && game.guessRemaining > 0 {
		/* It's still the current guesser's turn. */
		return true
	}
	return false
}

func ClueHandler(event Event, c *Client) error {
	game := c.game
	if game == nil {
		return fmt.Errorf("game does not exist")
	}
	if !game.active {
		return fmt.Errorf("inactive game")
	}

	// if we're here, a clue was given; now it's the guesser's turn
	game.roleTurn = guesser

	var clue GiveClueEvent
	if err := json.Unmarshal(event.Payload, &clue); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}
	if (clue.NumCards <= 0) {
		/* TODO: clue.NumCards == -1 if ChatGPT returned something
		   unparseable or barely parseable. Consider handling this
		   differently, e.g. have ChatGPT try again. */
		   
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

	// remove client from game, if there is one
	if c.game != nil {
		c.game.removePlayer(c.username)
	}

	// remove client from old chat room
	delete(c.manager.chats[oldroom], c.username)

	// notify old chat room that client has left
	c.manager.notifyClients(oldroom, EventExitRoom, event.Payload)

	// record whether there is a game in progress in the new room
	changeroom.GameInProgress = false
	if _, exists:= c.manager.games[newroom]; exists {
		if len(c.manager.games[newroom].players) > 0 {
			changeroom.GameInProgress = true
		}
	}

	// notify new room that the client is entering
	c.manager.notifyClients(newroom, EventEnterRoom, changeroom)

	// enter client into new chat room
	c.chatroom = newroom
	c.manager.makeChatRoom(newroom)
	c.manager.chats[newroom][c.username] = c

	// send list of current chat room participants to client
	changeroom.Participants = c.manager.chats[newroom].listClients()
	outgoingEvent, err := packageMessage(EventEnterRoom, changeroom)
	c.egress <- outgoingEvent
	return err
}

func TeamChangeHandler(event Event, c *Client) error {
	c.team = c.team.Change()
	updateMsg := PlayerAlignmentResponse {
		UserName: c.username,
		TeamColor: c.team,
		Role: c.role,
	}
	return c.manager.notifyClients(c.chatroom, EventUpdateParticipant, updateMsg)
}

func RoleChangeHandler(event Event, c *Client) error {
	c.role = c.role.Change()
	updateMsg := PlayerAlignmentResponse {
		UserName: c.username,
		TeamColor: c.team,
		Role: c.role,
	}
	return c.manager.notifyClients(c.chatroom, EventUpdateParticipant, updateMsg)
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

	return c.manager.notifyClients(c.chatroom, EventNewMessage, broadMessage)
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
	m.Lock()
	defer m.Unlock()

	if _, exists := m.chats[name]; exists {
		return
	}
	m.chats[name] = make(ClientList)
}

func (m *Manager) makeGame(name string, players ClientList, bots *BotActions) (*Game, error) {
	m.Lock()
	defer m.Unlock()

	if game, exists := m.games[name]; exists {
		return game, nil
	}
	actions := getActions(players, bots)
	if !actions.validate() {
		return nil, fmt.Errorf("invalid actions")
	}
	game := &Game {
		name: name,
		players: maps.Clone(players),
		cards: getCards(),
		actions: actions,
		teamTurn: red,
		roleTurn: cluegiver,
		score: Score {
			red: 9,
			blue: 8,
		},
		manager: m,
		active: true,
	}
	if game.actions.playerCount(red) == 0 {
		game.teamTurn = blue
	}
	game.makeBot(bots)
	m.games[name] = game
	
	for _, player := range players {
		player.game = game
	}

	return game, nil
}

/* Return true if game was deleted. */
func (m *Manager) removeGame(room string, message ...string) bool {
	m.Lock()
	defer m.Unlock()
	
	game := m.games[room]
	if game != nil {
		if game.active {
			gameOverMsg := GameOverEvent{
				Cards: game.cards.getUnrevealedCards(),
			}
			if len(message) > 0 {
				gameOverMsg.Message = message[0]
			}
			m.notifyClients(room, EventGameOver, gameOverMsg)
			game.active = false
			game.bot = nil
		}
		if len(game.players) == 0 {
			delete(m.games, room)
			return true
		}
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

	log.Info().Msg("new connection")

	// upgrade regular http connection into websocket
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("could not upgrade to websocket")
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
	}

	type response struct {
		OTP string `json:"otp"`
	}

	var (
		req  userLoginRequest
		resp response
	)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check for valid username (non-empty, no whitespace)
	if req.Username == "" {
		http.Error(w, "User name must not be empty", http.StatusBadRequest)
		return
	}
	if regexp.MustCompile(`\s`).MatchString(req.Username) {
		http.Error(w, "User name must not contain whitespace", http.StatusBadRequest)
		return
	}

	// Enforce unique usernames
	if _, exists := m.clients[req.Username]; exists {
		// someone with this username is already logged in
		http.Error(w, 
			"User name \"" + req.Username + "\" is already logged in. Choose a different username.",
			http.StatusConflict)
		return
	} else {
		otp := m.otps.NewOTP(req.Username)
		resp.OTP = otp.Key
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Error().Err(err)
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
		m.games[room].removePlayer(client.username)
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