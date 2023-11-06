package main

import (
	"context"
	"reflect"
	"testing"
)

func setupManager(t *testing.T) *Manager {
	t.Helper()

	ctx := context.Background()
	manager := NewManager(ctx)
	client := NewClient("testClient", nil, manager)
	manager.addClient(client)
	return manager
}

func setupGame(t *testing.T) *Manager {
	t.Helper()

	manager := setupManager(t)
	readDictionary("./codenames-wordlist.txt")
	manager.makeGame("test")
	manager.games["test"].players["testClient"] = manager.clients["testClient"]
	return manager
}

func TestAddClient(t *testing.T) {
	manager := setupManager(t)

	if _, exists := manager.chats[defaultChatRoom]; !exists {
		t.Error("default chat room does not exist")
	}

	if _, exists := manager.chats[defaultChatRoom]["testClient"]; !exists {
		t.Error("test client was not added to default chat room")
	}
}

func TestMakeGame(t *testing.T) {
	manager := setupGame(t)
	readDictionary("./codenames-wordlist.txt")

	manager.makeGame("test")
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