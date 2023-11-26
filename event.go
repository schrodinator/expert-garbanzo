package main

import (
	"encoding/json"
	"time"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error
type EventHandlerList map[string]EventHandler

const (
	EventSendMessage = "send_message"
	EventNewMessage  = "new_message"
	EventEnterRoom   = "enter_room"
	EventExitRoom    = "exit_room"
	EventChangeTeam  = "change_team"
	EventChangeRole  = "change_role"
	EventNewGame     = "new_game"
	EventMakeGuess   = "guess_event"
	EventGiveClue    = "give_clue"
	EventAbortGame   = "abort_game"
	EventEndTurn     = "end_turn"
)

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
	Color   string `json:"color"`
}

type NewMessageEvent struct {
	SendMessageEvent
	SentTime  time.Time `json:"sentTime"`
}

type ChangeRoomEvent struct {
	UserName  string `json:"clientName"`
	RoomName  string `json:"roomName"`
}

type NewGameRequestEvent struct {
	Bots BotList `json:"bots"`
}

type NewGameResponseEvent struct {
	Cards      map[string]string  `json:"cards"`
	SentTime   time.Time          `json:"sentTime"`
}

type AbortGameEvent struct {
	UserName  string `json:"clientName"`
	TeamColor Team   `json:"teamColor"`
}

type GiveClueEvent struct {
	Clue      string `json:"clue"`
	NumCards  int    `json:"numCards,string"`
	From      string `json:"from"`
	TeamColor Team   `json:"teamColor"`
}

type GuessEvent struct {
	Guess    string `json:"guess"`
	Guesser  string `json:"guesser"`
}

type EndTurnEvent struct {
	TeamTurn  Team `json:"teamTurn"`
	RoleTurn  Role `json:"roleTurn"`
}

type GuessResponseEvent struct {
	GuessEvent
	EndTurnEvent
	TeamColor      Team   `json:"teamColor"`
	CardColor      string `json:"cardColor"`
	Correct        bool   `json:"correct"`
	GuessRemaining int    `json:"guessRemaining"`
	Score          Score  `json:"score"`
}