package main

import (
	"reflect"
	"testing"
)

// Calling Change() on a Role should return a Role.
func TestRoleChangeType(t *testing.T) {
	var myRole Role
	myRole = cluegiver
	roleChange := myRole.Change()
	if reflect.TypeOf(roleChange) != reflect.TypeOf(myRole) {
		t.Errorf("type of roleChange: %v", reflect.TypeOf(roleChange))
	}
}

/* Calling Change() on a cluegiver Role should return
   the guesser Role. */
func TestRoleChangeValue0(t *testing.T) {
	myRole := cluegiver
	roleChange := myRole.Change()
	if roleChange != guesser {
		t.Errorf("value of roleChange: %v", roleChange)
	}
}

/* Calling Change() on a guesser Role should return
   the cluegiver Role. */
func TestRoleChangeValue1(t *testing.T) {
	myRole := guesser
	roleChange := myRole.Change()
	if roleChange != cluegiver {
		t.Errorf("value of roleChange: %v", roleChange)
	}
}

// Calling Change() on a Team should return a Team.
func TestTeamChangeType(t *testing.T) {
	var myTeam Team
	myTeam = red
	teamChange := myTeam.Change()
	if reflect.TypeOf(teamChange) != reflect.TypeOf(myTeam) {
		t.Errorf("type of teamChange is %v", reflect.TypeOf(teamChange))
	}
}

// Calling Change() on the red Team should return blue.
func TestTeamChangeValue0(t *testing.T) {
	myTeam := red
	teamChange := myTeam.Change()
	if teamChange != blue {
		t.Errorf("value of teamChange is %v", teamChange)
	}
}

// Calling Change() on the blue Team should return red.
func TestTeamChangeValue1(t *testing.T) {
	myTeam := blue
	teamChange := myTeam.Change()
	if teamChange != red {
		t.Errorf("value of teamChange is %v", teamChange)
	}
}

// Calling changeTurn() with red cluegiver should set red guesser.
func TestChangeTurnRedCluegiver(t *testing.T) {
	var game Game
	game.roleTurn = cluegiver
	game.teamTurn = red
	game.teamTurn, game.roleTurn = changeTurn(game.teamTurn, game.roleTurn)
	if game.roleTurn != guesser {
		t.Errorf("role after changeTurn is %v", game.roleTurn)
	}
	if game.teamTurn != red {
		t.Errorf("team after changeTurn is %v", game.teamTurn)
	}
}

// Calling changeTurn() with red guesser should set blue cluegiver.
func TestChangeTurnRedGuesser(t *testing.T) {
	var game Game
	game.roleTurn = guesser
	game.teamTurn = red
	game.teamTurn, game.roleTurn = changeTurn(game.teamTurn, game.roleTurn)
	if game.roleTurn != cluegiver {
		t.Errorf("role after changeTurn is %v", game.roleTurn)
	}
	if game.teamTurn != blue {
		t.Errorf("team after changeTurn is %v", game.teamTurn)
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
	manager := setupGame(t)
	game := manager.games["test"]

	if game.score[red] != 9 || game.score[blue] != 8 {
		t.Errorf("problem in setup: red score is %v and blue score is %v",
	             game.score[red], game.score[blue])
	}

	type test struct {
		cardColor   string
		expectScore Score
	}
	tests := []test{
		{ cardColor: "red", expectScore: Score{red: 8, blue: 8} },
		{ cardColor: "blue", expectScore: Score{red: 8, blue: 7} },
		{ cardColor: "neutral", expectScore: Score{red: 8, blue: 7} },
		{ cardColor: deathCard, expectScore: Score{red: 8, blue: 7} },
		{ cardColor: "red", expectScore: Score{red: 7, blue: 7} },
	}

	for _, tt := range tests {
		game.updateScore(tt.cardColor)
		if !reflect.DeepEqual(game.score, tt.expectScore) {
			t.Fatalf("test %v: expected: %v, got: %v",
			         tt.cardColor, tt.expectScore, game.score)
		}
	}
}