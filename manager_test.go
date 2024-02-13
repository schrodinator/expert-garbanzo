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
	client1 := NewClient("testClient1", ws, manager)
	manager.addClient(client1)
	client2 := NewClient("testClient2", nil, manager)
	manager.addClient(client2)
	return manager
}

func setupGame(t *testing.T, ws *websocket.Conn, bots *BotActions) *Manager {
	t.Helper()

	manager := setupManager(t, ws)
	readWordList("./wordlist.txt")
	client1 := manager.clients["testClient1"]
	client2 := manager.clients["testClient2"]
	client1.chatroom = "test"
	client2.chatroom = "test"
	client2.role = cluegiver
	manager.makeGame("test", ClientList{"testClient1": client1, "testClient2": client2}, bots)
	
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

func setupWSTest(t *testing.T, ws *websocket.Conn, bots *BotActions) *Manager {
	t.Helper()

	manager := setupDeck(t, ws, bots)
	// testClient2 has no websocket. Must be removed for WS test.
	manager.games["test"].removePlayer("testClient2")
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
		t.Errorf("test game does not exist")
	}
	game := manager.games["test"]

	if reflect.TypeOf(game.players) != reflect.TypeOf(ClientList{}) {
		t.Errorf("'players' is type %T, not type ClientList", game.players)
	}

	if _, exists := game.players["testClient1"]; !exists {
		t.Error("could not add client to 'players'")
	}

	if len(game.cards) != totalNumCards {
		t.Errorf("not dealing with a full deck: %v cards", len(game.cards))
	}

	if game.score[red] != 9 {
		t.Errorf("initial score for red team is %v", game.score[red])
	}

	if game.teamTurn != red {
		t.Error("initial team turn is not red")
	}

	if game.roleTurn != cluegiver {
		t.Error("initial role turn is not cluegiver")
	}
}

func TestGuessEvaluationHandler(t *testing.T) {
	s, ws := setupWSTestServer(t)
	defer s.Close()
	defer ws.Close()
	
	manager := setupWSTest(t, ws, nil)
	game := manager.games["test"]
	game.roleTurn = guesser
	game.guessRemaining = totalNumCards
	manager.games["test"] = game
	client := manager.clients["testClient1"]
	client.game = game

	go client.writeMessages()

	guess := GuessEvent{
		Guess: "redword",
		Guesser: "testClient1",
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
		GuessEvent: GuessEvent{Guess: "redword", Guesser: "testClient1",},
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
}