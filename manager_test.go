package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func setupManager(t *testing.T, ws *websocket.Conn) *Manager {
	t.Helper()

	ctx := context.Background()
	manager := NewManager(ctx)
	client := NewClient("testClient", ws, manager)
	manager.addClient(client)
	return manager
}

func setupGame(t *testing.T, ws *websocket.Conn, bots *BotActions) *Manager {
	t.Helper()

	manager := setupManager(t, ws)
	readDictionary("./codenames-wordlist.txt")
	client := manager.clients["testClient"]
	client.chatroom = "test"
	manager.makeGame("test", ClientList{"testClient": client}, bots)
	
	return manager
}

func setupDeck(t *testing.T, ws *websocket.Conn, bots *BotActions) *Manager {
	t.Helper()

	manager := setupGame(t, ws, bots)
	game := manager.games["test"]
	game.cards = Deck{
		"redword": "red",
		"blueword": "blue",
		"neutralword": "neutral",
		"deathword": deathCard,
	}
	return manager
}

func setupWSTestServer(t *testing.T) (*httptest.Server, *websocket.Conn) {
	t.Helper()

	echo := func(w http.ResponseWriter, r *http.Request) {
		websocketUpgrader.CheckOrigin = func(r *http.Request) bool { return true }
		c, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}

    // Create test server with the echo handler.
    s := httptest.NewServer(http.HandlerFunc(echo))

    // Replace "http" with "ws" in the URL string.
    u := "ws" + strings.TrimPrefix(s.URL, "http")

    // Connect to the server
    ws, _, err := websocket.DefaultDialer.Dial(u, nil)
    if err != nil {
        t.Fatalf("%v", err)
    }

	return s, ws
}

func TestMakeGame(t *testing.T) {
	manager := setupGame(t, nil, nil)

	if _, exists := manager.games["test"]; !exists {
		t.Error("test game does not exist")
	}

	var cl ClientList
	if reflect.TypeOf(manager.games["test"].players) != reflect.TypeOf(cl) {
		t.Errorf("'players' is type %T, not type ClientList", manager.games["test"].players)
	}

	if _, exists := manager.games["test"].players["testClient"]; !exists {
		t.Error("could not add client to 'players'")
	}

	if len(manager.games["test"].cards) != totalNumCards {
		t.Errorf("not dealing with a full deck: %v cards", len(manager.games["test"].cards))
	}

	if manager.games["test"].score[red] != 9 {
		t.Errorf("initial score for red team is %v", manager.games["test"].score[red])
	}
}

func TestGuessEvaluationHandler(t *testing.T) {
	s, ws := setupWSTestServer(t)
	manager := setupDeck(t, ws, nil)
	game := manager.games["test"]
	game.roleTurn = guesser
	game.guessRemaining = totalNumCards
	manager.games["test"] = game
	client := manager.clients["testClient"]
	client.game = game

	go client.writeMessages()

	guess := GuessEvent{
		Guess: "redword",
		Guesser: "testClient",
	}
	payload, err := json.Marshal(guess)
	if err != nil {
		t.Fatalf("could not marshal guess: %v", err)
	}
	event := Event{
		Type: EventMakeGuess,
		Payload: payload,
	}

	GuessEvaluationHandler(event, client)

	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("could not read message: %v", err)
	}
	var e Event
	if err := json.Unmarshal(message, &e); err != nil {
		t.Fatalf("could not unmarshal message: %v", err)
	}
	if e.Type != EventMakeGuess {
		t.Errorf("wrong Type: %v", e.Type)
	}
	expect := GuessResponseEvent{
		GuessEvent: GuessEvent{Guess: "redword", Guesser: "testClient",},
		EndTurnEvent: EndTurnEvent{TeamTurn: red, RoleTurn: guesser,},
		TeamColor: red,
		CardColor: "red",
		Correct: true,
		GuessRemaining: 25,
		Score: Score{red:8, blue: 8,},
	}
	expectJSON, err := json.Marshal(expect)
	if err != nil {
		t.Fatalf("could not marshal 'expect': %v", err)
	}
	if !bytes.Equal(e.Payload, expectJSON) {
		t.Errorf("byte comparison payload not equal to expected")
	}
	var gre GuessResponseEvent
	if err := json.Unmarshal(e.Payload, &gre); err != nil {
		t.Fatalf("could not unmarshal message: %v", err)
	}
	if !reflect.DeepEqual(gre, expect) {
		t.Errorf("Expected: %#v\nGot: %#v", expect, gre)
	}

	ws.Close()
	s.Close()
}