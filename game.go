package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
)

var (
	dictionary []string
	dictLen    int
)

type Team int
const (
	red  Team = iota
	blue
)
func (t Team) String() string {
	return [...]string{"red", "blue"}[t]
}
func (t Team) EnumIndex() int {
	return int(t)
}
func (t Team) Change() Team {
	if t == 0 {
		return Team(1)
	}
	return Team(0)
}
func GetTeam(s string) (Team, error) {
	for _, team := range([2]Team{red, blue}) {
		if s == team.String() {
			return team, nil
		}
	}
	return -1, fmt.Errorf("no Team match for input %v", s)
}

type Role int
const (
	cluegiver Role = iota
	guesser
)
func (r Role) String() string {
	return [...]string{"cluegiver", "guesser"}[r]
}
func (r Role) EnumIndex() int {
	return int(r)
}
func (r Role) Change() Role {
	if r == 0 {
		return Role(1)
	}
	return Role(0)
}

type Score   map[Team]int
func (score Score) MarshalJSON() ([]byte, error) {
	s := make(map[string]int)
	for t, i := range score {
		s[t.String()] = i
	}
	return json.Marshal(s)
}

type GameMap map[string]Game
type Deck    map[string]string

type Game struct {
	players         ClientList
	cards           Deck
	teamTurn        Team
	roleTurn        Role
	guessRemaining  int
	score           Score
}

const totalNumCards = 25


func changeTurn(team Team, role Role) (Team, Role) {
	newTeam := team
	newRole := role.Change()
	if newRole == cluegiver {
		newTeam = team.Change()
	}
	return newTeam, newRole
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
	fmt.Printf("dictLen: %v", dictLen)
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

func (game Game) updateScore(cardColor string) {
	team, err := GetTeam(cardColor)
	if err != nil {
		// a neutral card or the death card
		return
	}
	game.score[team] -= 1
}