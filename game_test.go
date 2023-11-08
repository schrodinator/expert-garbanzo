package main

import (
	"reflect"
	"testing"
)

// Change Role from cluegiver to guesser to cluegiver
func TestRoleChangeType(t *testing.T) {
	var myRole Role
	myRole = cluegiver
	roleChange := myRole.Change()
	if reflect.TypeOf(roleChange) != reflect.TypeOf(myRole) {
		t.Errorf("type of roleChange: %v", reflect.TypeOf(roleChange))
	}
	if roleChange != guesser {
		t.Errorf("value of first roleChange: %v", roleChange)
	}

	roleChange = roleChange.Change()
	if roleChange != cluegiver {
		t.Errorf("value of second roleChange: %v", roleChange)
	}
}

// Change Team from red to blue to red
func TestTeamChange(t *testing.T) {
	var myTeam Team
	myTeam = red

	teamChange := myTeam.Change()
	if reflect.TypeOf(teamChange) != reflect.TypeOf(myTeam) {
		t.Errorf("type of first teamChange is %v", reflect.TypeOf(teamChange))
	}
	if teamChange != blue {
		t.Errorf("value of first teamChange is %v", teamChange)
	}

	teamChange = teamChange.Change()
	if teamChange != red {
		t.Errorf("value of second teamChange is %v", teamChange)
	}
}

// Turns should go (red cluegiver) -> (red guesser) -> (blue cluegiver)
func TestChangeTurn(t *testing.T) {
	var game Game
	game.roleTurn = cluegiver
	game.teamTurn = red

	game.teamTurn, game.roleTurn = changeTurn(game.teamTurn, game.roleTurn)
	if game.roleTurn != guesser {
		t.Errorf("role after first changeTurn is %v", game.roleTurn)
	}
	if game.teamTurn != red {
		t.Errorf("team after first changeTurn is %v", game.teamTurn)
	}

	game.teamTurn, game.roleTurn = changeTurn(game.teamTurn, game.roleTurn)
	if game.roleTurn != cluegiver {
		t.Errorf("role after second changeTurn is %v", game.roleTurn)
	}
	if game.teamTurn != blue {
		t.Errorf("team after second changeTurn is %v", game.teamTurn)
	}
}

/* Calling getCards() returns 25 cards.
   The function is nondeterministic so this is not a perfect test. */
func TestGetCards(t *testing.T) {
	readDictionary("./codenames-wordlist.txt")
	cards := getCards()
	if len(cards) != totalNumCards {
		t.Errorf("not dealing with a full deck: %v cards", len(cards))
	}
}

func TestUpdateScore(t *testing.T) {
	manager := setupGame(t, nil)
	game := manager.games["test"]

	if game.score[red] != 9 || game.score[blue] != 8 {
		t.Errorf("problem in setup: red score is %v and blue score is %v",
	             game.score[red], game.score[blue])
	}

	type test struct {
		name        string
		cardColor   string
		expectScore Score
	}
	tests := []test{
		{ name: "red", cardColor: "red", expectScore: Score{red: 8, blue: 8} },
		{ name: "blue", cardColor: "blue", expectScore: Score{red: 8, blue: 7} },
		{ name: "neutral", cardColor: "neutral", expectScore: Score{red: 8, blue: 7} },
		{ name: "death card", cardColor: deathCard, expectScore: Score{red: 8, blue: 7} },
		{ name: "second red", cardColor: "red", expectScore: Score{red: 7, blue: 7} },
	}

	for _, tt := range tests {
		game.updateScore(tt.cardColor)
		if !reflect.DeepEqual(game.score, tt.expectScore) {
			t.Fatalf("test %v: expected: %v, got: %v",
			         tt.name, tt.expectScore, game.score)
		}
	}
}