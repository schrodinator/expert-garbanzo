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

	game.changeTurn()
	if game.roleTurn != guesser {
		t.Errorf("role after first changeTurn is %v", game.roleTurn)
	}
	if game.teamTurn != red {
		t.Errorf("team after first changeTurn is %v", game.teamTurn)
	}

	game.changeTurn()
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

type guesstest struct {
	name           string
	cardColor      string
	expectCorrect  bool
	expectScore    Score
	expectGuess    int
	expectTeamTurn Team
	expectRoleTurn Role
}

func TestEvaluateGuess1(t *testing.T) {
	manager := setupDeck(t, nil)
	game := manager.games["test"]
	game.teamCounts[red] = 2
	game.teamCounts[blue] = 2
	game.roleTurn = guesser
	game.guessRemaining = 3

	var guesses = []guesstest {
		{
			name: "red1",
			cardColor: "red",
			expectCorrect: true,
			expectScore: Score{red: 8, blue: 8},
			expectGuess: 2,
			expectTeamTurn: red,
			expectRoleTurn: guesser,
		},
		{
			name: "red2",
			cardColor: "red",
			expectCorrect: true,
			expectScore: Score{red: 7, blue: 8},
			expectGuess: 1,
			expectTeamTurn: red,
			expectRoleTurn: guesser,
		},
		{
			name: "red3",
			cardColor: "red",
			expectCorrect: true,
			expectScore: Score{red: 6, blue: 8},
			expectGuess: 0,
			expectTeamTurn: blue,
			expectRoleTurn: cluegiver,
		},
	}

	for _, tt := range guesses {
		correct := game.evaluateGuess(tt.cardColor)
		if correct != tt.expectCorrect {
			t.Errorf("test %v, correct: expected: %v, got: %v",
					 tt.name, tt.expectCorrect, correct)
		}
		if !reflect.DeepEqual(game.score, tt.expectScore) {
			t.Errorf("test %v, score: expected: %v, got: %v",
			         tt.name, tt.expectScore, game.score)
		}
		if game.guessRemaining != tt.expectGuess {
			t.Errorf("test %v, guesses remaining: expected: %v, got: %v",
			         tt.name, tt.expectGuess, game.guessRemaining)
		}
		if game.teamTurn != tt.expectTeamTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectTeamTurn, game.teamTurn)
		}
		if game.roleTurn != tt.expectRoleTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectRoleTurn, game.roleTurn)
		}
	}
}

func TestEvaluateGuess2(t *testing.T) {
	manager := setupDeck(t, nil)
	game := manager.games["test"]
	game.teamCounts[red] = 2
	game.teamCounts[blue] = 2
	game.roleTurn = guesser
	game.guessRemaining = 3

	var guesses = []guesstest {
		{
			name: "red1",
			cardColor: "red",
			expectCorrect: true,
			expectScore: Score{red: 8, blue: 8},
			expectGuess: 2,
			expectTeamTurn: red,
			expectRoleTurn: guesser,
		},
		{
			name: "blue2",
			cardColor: "blue",
			expectCorrect: false,
			expectScore: Score{red: 8, blue: 7},
			expectGuess: 0,
			expectTeamTurn: blue,
			expectRoleTurn: cluegiver,
		},
	}

	for _, tt := range guesses {
		correct := game.evaluateGuess(tt.cardColor)
		if correct != tt.expectCorrect {
			t.Errorf("test %v, correct: expected: %v, got: %v",
					 tt.name, tt.expectCorrect, correct)
		}
		if !reflect.DeepEqual(game.score, tt.expectScore) {
			t.Errorf("test %v, score: expected: %v, got: %v",
			         tt.name, tt.expectScore, game.score)
		}
		if game.guessRemaining != tt.expectGuess {
			t.Errorf("test %v, guesses remaining: expected: %v, got: %v",
			         tt.name, tt.expectGuess, game.guessRemaining)
		}
		if game.teamTurn != tt.expectTeamTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectTeamTurn, game.teamTurn)
		}
		if game.roleTurn != tt.expectRoleTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectRoleTurn, game.roleTurn)
		}
	}
}

func TestEvaluateGuess3(t *testing.T) {
	manager := setupDeck(t, nil)
	game := manager.games["test"]
	game.teamCounts[red] = 2
	game.teamCounts[blue] = 2
	game.roleTurn = guesser
	game.guessRemaining = 25

	var guesses = []guesstest {
		{
			name: "red1",
			cardColor: "red",
			expectCorrect: true,
			expectScore: Score{red: 8, blue: 8},
			expectGuess: 25,
			expectTeamTurn: red,
			expectRoleTurn: guesser,
		},
		{
			name: "neutral2",
			cardColor: "neutral",
			expectCorrect: false,
			expectScore: Score{red: 8, blue: 8},
			expectGuess: 0,
			expectTeamTurn: blue,
			expectRoleTurn: cluegiver,
		},
	}

	for _, tt := range guesses {
		correct := game.evaluateGuess(tt.cardColor)
		if correct != tt.expectCorrect {
			t.Errorf("test %v, correct: expected: %v, got: %v",
					 tt.name, tt.expectCorrect, correct)
		}
		if !reflect.DeepEqual(game.score, tt.expectScore) {
			t.Errorf("test %v, score: expected: %v, got: %v",
			         tt.name, tt.expectScore, game.score)
		}
		if game.guessRemaining != tt.expectGuess {
			t.Errorf("test %v, guesses remaining: expected: %v, got: %v",
			         tt.name, tt.expectGuess, game.guessRemaining)
		}
		if game.teamTurn != tt.expectTeamTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectTeamTurn, game.teamTurn)
		}
		if game.roleTurn != tt.expectRoleTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectRoleTurn, game.roleTurn)
		}
	}
}

func TestEvaluateGuess4(t *testing.T) {
	manager := setupDeck(t, nil)
	game := manager.games["test"]
	game.teamCounts[red] = 2
	game.teamCounts[blue] = 2
	game.roleTurn = guesser
	game.guessRemaining = 25

	var guesses = []guesstest {
		{
			name: "death card",
			cardColor: deathCard,
			expectCorrect: false,
			expectScore: Score{red: 9, blue: 8},
			expectGuess: 0,
			expectTeamTurn: blue,
			expectRoleTurn: cluegiver,
		},
	}

	for _, tt := range guesses {
		correct := game.evaluateGuess(tt.cardColor)
		if correct != tt.expectCorrect {
			t.Errorf("test %v, correct: expected: %v, got: %v",
					 tt.name, tt.expectCorrect, correct)
		}
		if !reflect.DeepEqual(game.score, tt.expectScore) {
			t.Errorf("test %v, score: expected: %v, got: %v",
			         tt.name, tt.expectScore, game.score)
		}
		if game.guessRemaining != tt.expectGuess {
			t.Errorf("test %v, guesses remaining: expected: %v, got: %v",
			         tt.name, tt.expectGuess, game.guessRemaining)
		}
		if game.teamTurn != tt.expectTeamTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectTeamTurn, game.teamTurn)
		}
		if game.roleTurn != tt.expectRoleTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectRoleTurn, game.roleTurn)
		}
	}
}

// Simulate a game with only one team
func TestEvaluateGuess5(t *testing.T) {
	manager := setupDeck(t, nil)
	game := manager.games["test"]
	game.teamCounts[red] = 2
	game.roleTurn = guesser
	game.guessRemaining = 25

	var guesses = []guesstest {
		{
			name: "blue1",
			cardColor: "blue",
			expectCorrect: false,
			expectScore: Score{red: 9, blue: 7},
			expectGuess: 0,
			expectTeamTurn: red,
			expectRoleTurn: cluegiver,
		},
	}

	for _, tt := range guesses {
		correct := game.evaluateGuess(tt.cardColor)
		if correct != tt.expectCorrect {
			t.Errorf("test %v, correct: expected: %v, got: %v",
					 tt.name, tt.expectCorrect, correct)
		}
		if !reflect.DeepEqual(game.score, tt.expectScore) {
			t.Errorf("test %v, score: expected: %v, got: %v",
			         tt.name, tt.expectScore, game.score)
		}
		if game.guessRemaining != tt.expectGuess {
			t.Errorf("test %v, guesses remaining: expected: %v, got: %v",
			         tt.name, tt.expectGuess, game.guessRemaining)
		}
		if game.teamTurn != tt.expectTeamTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectTeamTurn, game.teamTurn)
		}
		if game.roleTurn != tt.expectRoleTurn {
			t.Errorf("test %v, team turn: expected: %v, got: %v",
			         tt.name, tt.expectRoleTurn, game.roleTurn)
		}
	}
}