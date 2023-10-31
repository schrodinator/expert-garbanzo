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
)

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
	Color   string `json:"color"`
}

type NewMessageEvent struct {
	SendMessageEvent
	Sent time.Time `json:"sentDate"`
}

type ChangeRoomEvent struct {
	Name string `json:"name"`
}

type NewGameEvent struct {
	Words map[string]string  `json:"wordsToAlignment"`
	Sent  time.Time          `json:"sentTime"`
}

type GuessEvent struct {
	Guess   string `json:"guess"`
	Guesser string `json:"guesser"`
}

type GuessResponseEvent struct {
	GuessEvent
	GuesserTeam    string `json:"guesserTeamColor"`
	CardAlignment  string `json:"cardAlignment"`
	Correct        bool   `json:"correct"`
}