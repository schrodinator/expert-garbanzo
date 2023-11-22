package main

import (
	"reflect"
	"strings"
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

func TestGetClueWords(t *testing.T) {
	deck := Deck{
		"word1": "red",
		"word2": "blue",
		"word3": "red",
		"word4": "blue",
		"word5": "neutral",
		"word6": deathCard,
		"word7": "guessed-blue",
		"word8": "guessed-red",
	}

	w := deck.getClueWords("blue")

	if len(w.myTeam) == 0 {
		t.Error("myTeam has length 0")
	}
	if len(w.others) == 0 {
		t.Error("others has length 0")
	}

	/* map keys can be returned in any order */
	if !(w.myTeam == "word2, word4" || w.myTeam == "word4, word2") {
		t.Errorf("myTeam word list: %v", w.myTeam)
	}

	for _, e := range [3]string{"word2", "word7", "word8"} {
		if strings.Contains(w.others, e) {
			t.Errorf("others should not contain %v: got %v", e, w.others)
		}
	}

	for _, e := range [4]string{"word1", "word3", "word5", "word6"} {
		if !strings.Contains(w.others, e) {
			t.Errorf("missing word %v: got %v", e, w.others)
		}
	}
}

func TestGetGuessWords(t *testing.T) {
	deck := Deck{
		"word1": "red",
		"word2": "blue",
		"word3": "red",
		"word4": "blue",
		"word5": "neutral",
		"word6": deathCard,
		"word7": "guessed-blue",
		"word8": "guessed-red",
	}

	w := deck.getGuessWords()

	if len(w) == 0 {
		t.Error("Guess words has length 0")
	}

	/* map keys can be returned in any order */
	for _, e := range [2]string{"word7", "word8"} {
		if strings.Contains(w, e) {
			t.Errorf("word list should not contain %v: got %v", e, w)
		}
	}

	for _, e := range [6]string{"word1", "word2", "word3", "word4", "word5", "word6"} {
		if !strings.Contains(w, e) {
			t.Errorf("missing word %v: got %v", e, w)
		}
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