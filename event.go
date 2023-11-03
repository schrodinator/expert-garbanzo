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
	EventChangeRoom  = "change_room"
	EventChangeTeam  = "change_team"
	EventChangeRole  = "change_role"
	EventNewGame     = "new_game"
	EventMakeGuess   = "guess_event"
	EventGiveClue    = "give_clue"
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
	UserName  string `json:"username"`
	RoomName  string `json:"roomname"`
}

type NewGameEvent struct {
	Cards      map[string]string  `json:"cards"`
	SentTime   time.Time          `json:"sentTime"`
}

type GuessEvent struct {
	Guess   string `json:"guess"`
	Guesser string `json:"guesser"`
}

type GuessResponseEvent struct {
	GuessEvent
	GuesserTeam  string `json:"guesserTeamColor"`
	CardColor    string `json:"cardColor"`
	Correct      bool   `json:"correct"`
	TeamTurn     string `json:"teamTurn"`
	RoleTurn     string `json:"roleTurn"`
}