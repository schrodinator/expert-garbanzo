package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

const totalNumCards = 25

var (
	dictionary []string
	dictLen    int
)

type Team string
const (
	red  Team = "red"
	blue Team = "blue"
)
func (t Team) String() string {
	return string(t)
}
func (t Team) Change() Team {
	switch t {
	case red:
		return blue
	case blue:
		return red
	default:
		return t
	}
}
func (t Team) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}
func (t Team) UnmarshalJSON(data []byte) error {
	var teamString string
	if err := json.Unmarshal(data, &teamString); err != nil {
		return err
	}
	t, err := NewTeam(teamString)
	return err
}
func NewTeam(s string) (Team, error) {
	switch s {
	case "red":
		return red, nil
	case "blue":
		return blue, nil
	default:
		return "", fmt.Errorf("invalid team: %s", s)
	}
}

type Role string
const (
	cluegiver Role = "cluegiver"
	guesser   Role = "guesser"
)
func (r Role) String() string {
	return string(r)
}
func (r Role) Change() Role {
	switch r {
	case guesser:
		return cluegiver
	case cluegiver:
		return guesser
	default:
		return r
	}
}
func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}
func (r Role) UnmarshalJSON(data []byte) error {
	var roleString string
	if err := json.Unmarshal(data, &roleString); err != nil {
		return err
	}
	r, err := NewRole(roleString)
	return err
}
func NewRole(r string) (Role, error) {
	switch r {
	case "cluegiver":
		return cluegiver, nil
	case "guesser":
		return guesser, nil
	default:
		return "", fmt.Errorf("invalid role: %s", r)
	}
}

type Deck map[string]string

type clueWords struct {
	myTeam string
	others string
}
func (d *Deck) getClueWords(team string) *clueWords {
	var myTeam []string
	var others []string
	for card, color := range *d {
		if color == team {
			myTeam = append(myTeam, card)
		} else if !strings.HasPrefix(color, "guess") {
			others = append(others, card)
		}
	}
	return &clueWords {
		myTeam: strings.Join(myTeam, ", "),
		others: strings.Join(others, ", "),
	}
}

func (d *Deck) getGuessWords() string {
	var words []string
	for card, color := range *d {
		if !strings.HasPrefix(color, "guess") {
			words = append(words, card)
		}
	}
	return strings.Join(words, ", ")
}

type GameList map[string]*Game
type Score    map[Team]int
type BotList  map[Team]map[Role]bool

type Game struct {
	players         ClientList
	bots            *BotList
	cards           Deck
	teamTurn        Team
	teamCounts      map[Team]int
	roleTurn        Role
	guessRemaining  int
	score           Score
	bot             *Bot
}

func (game *Game) changeTurn() {
	game.roleTurn = game.roleTurn.Change()
	if len(game.teamCounts) == 1 {
		return
	}
	if game.roleTurn == cluegiver {
		game.teamTurn = game.teamTurn.Change()
	}
}

func (game *Game) updateScore(cardColor string) {
	team, err := NewTeam(cardColor)
	if err != nil {
		// cardColor is not red or blue
		return
	}
	game.score[team] -= 1
}

func (game *Game) updateGuessesRemaining(correct bool) {
	if !correct {
		game.guessRemaining = 0
		return
	}
	if game.guessRemaining < totalNumCards {
		game.guessRemaining -= 1
	}
}

func (game *Game) evaluateGuess(cardColor string) bool {
	game.updateScore(cardColor)
	correct := cardColor == game.teamTurn.String()
	game.updateGuessesRemaining(correct)
	if game.guessRemaining <= 0 {
		game.changeTurn()
	}
	return correct
}

func readDictionary(filePath string) error {
	if len(dictionary) == 0 {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Failed to open file at path %v: %v", filePath, err)
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			word := scanner.Text()
			dictionary = append(dictionary, word)
		}
		dictLen = len(dictionary)

		if err := scanner.Err(); err != nil {
			fmt.Println("Failed to parse dictionary:", err)
			return err
		}
	}

	return nil
}

func getCards() Deck {
	var colors = [25]string{
		"red", "red", "red", "red", "red", "red", "red", "red", "red",
	    "blue", "blue", "blue", "blue", "blue", "blue", "blue", "blue",
	    "black",
	    "neutral", "neutral", "neutral", "neutral", "neutral", "neutral", "neutral"}
	cards := make(Deck, totalNumCards)
	for i := 0; i < totalNumCards; i++ {
		word := dictionary[rand.Intn(dictLen)]
		// ensure each word is unique
		if _, exists := cards[word]; exists {
			i--
			continue
		}
		cards[word] = colors[i]
	}
	return cards
}

func whiteCards(deck Deck) Deck {
	whiteDeck := make(Deck, totalNumCards)
	for card := range deck {
		whiteDeck[card] = "white"
	}
	return whiteDeck
}