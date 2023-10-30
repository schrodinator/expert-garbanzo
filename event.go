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

const (
	EventSendMessage = "send_message"
	EventNewMessage  = "new_message"
	EventChangeRoom  = "change_room"
	EventNewGame     = "new_game"
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
	Words map[string]string  `json:"words"`
	Sent  time.Time `json:"sentTime"`
}