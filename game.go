package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
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
func (t Team) Title() string {
	switch t {
	case red:
		return "Red"
	case blue:
		return "Blue"
	default:
		return ""
	}
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

type Deck map[string]string

type clueWords struct {
	myTeam string
	others string
}
func (d Deck) getClueWords(team Team) *clueWords {
	var myTeam []string
	var others []string
	for card, color := range d {
		if color == team.String() {
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

func (d Deck) getGuessWords() string {
	var words []string
	for card, color := range d {
		if !strings.HasPrefix(color, "guess") {
			words = append(words, card)
		}
	}
	return strings.Join(words, ", ")
}

func (d Deck) whiteCards() Deck {
	whiteDeck := make(Deck, totalNumCards)
	for card := range d {
		whiteDeck[card] = "white"
	}
	return whiteDeck
}

type Actions  map[Team]map[Role]int
func (actions Actions) teamCount() int {
	ct := 0
	for _, t := range []Team{ red, blue } {
		if actions.playerCount(t) > 0 {
			ct++
		}
	}
	return ct
}
func (actions Actions) playerCount(team Team) int {
	return actions[team][guesser] + actions[team][cluegiver]
}
func (actions Actions) validate() bool {
	for _, t := range []Team{ red, blue } {
		if (actions[t][cluegiver] == 0 && actions[t][guesser] != 0) ||
		   (actions[t][cluegiver] != 0 && actions[t][guesser] == 0) {
			/* Team does not have both a guesser and a cluegiver.
			   Allow for a team with neither (i.e. single-team co-op game). */
			return false
		}
	}
	return true
}

type GameList map[string]*Game
type Score    map[Team]int

type Game struct {
	name            string
	players         ClientList
	cards           Deck
	teamTurn        Team
	actions         Actions
	playerActions   Actions
	roleTurn        Role
	guessRemaining  int
	score           Score
	bot             *Bot
	manager         *Manager
	active          bool
}

func (game *Game) notifyPlayers(messageType string, message any) error {
	outgoingEvent, err := packageMessage(messageType, message)
	if err != nil {
		return err
	}

	for _, client := range game.players {
		client.egress <- outgoingEvent
	}

	return nil
}

func (game *Game) notifySomePlayers(team Team, role Role, messageType string, message any) error {
	outgoingEvent, err := packageMessage(messageType, message)
	if err != nil {
		return err
	}

	for _, client := range game.players {
		if client.team == team && client.role == role {
			client.egress <- outgoingEvent
		}
	}

	return nil
}

func (game *Game) changeTurn() {
	game.roleTurn = game.roleTurn.Change()
	if game.actions.teamCount() == 1 {
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

func (game *Game) makeBot(ba *BotActions) {
	if ba != nil &&
	   (ba.hasAction(cluegiver) || ba.hasAction(guesser)) {
		game.bot = NewBot(game, ba)
	}
}

func (game *Game) botPlay(clue GiveClueEvent) error {
	if game.bot == nil {
		return nil
	}
	eventType, clueStruct, team, role := game.bot.Play(clue)
	if eventType == "" || clueStruct == nil {
		// TODO: better handling of missing bot response
		return nil
	}
	/* If human players share this role, tell them the bot's
	   suggestion. Do not play for them. */
	if game.playerActions[team][role] > 0 {
		message := NewMessageEvent {
			SentTime: time.Now(),
			SendMessageEvent: SendMessageEvent {
				Message: clueStruct.response,
				From: "ChatBot " + role.String(),
				Color: team.String(),
			},
		}
		game.notifySomePlayers(team, role, EventNewMessage, message)
		return nil
	}
	switch eventType {
	case EventMakeGuess:
		/* The bot could/should return multiple guesses. */
		for _, guess := range clueStruct.capsWords {
			if _, exists := game.cards[guess]; !exists {
				continue
			}
			guessResponse := GuessResponseEvent {
				GuessEvent: GuessEvent{
					Guess: guess,
					Guesser: "ChatBot",
				},
				TeamColor: team,
			}
			if !GuessEvaluation(guessResponse, game.bot.client) {
				/* Incorrect guess, or game over. */
				if game != nil && game.active {
					return game.botPlay(GiveClueEvent{})
				}
				return nil
			}
		}
		/* We could get here if the bot returns fewer guesses
		   than requested, or if we could not parse the guess
		   words from the response. In this case, we need to
		   end the bot's turn in order to continue the game. */
		if game.teamTurn == game.bot.client.team &&
		   game.roleTurn == game.bot.client.role {
			EndTurnHandler(Event{}, game.bot.client)
		}
		return nil

	case EventGiveClue:
		e := GiveClueEvent {
			Clue: clueStruct.word,
			NumCards: clueStruct.numGuess,
			From: "ChatBot",
			TeamColor: game.teamTurn,
		}
		evt, err := packageMessage(eventType, e)
		if err != nil {
			return err
		}
		ClueHandler(evt, game.bot.client)
		return nil

	default:
		return fmt.Errorf("unknown event type: %v", eventType)
	}
}

func (game *Game) validGame() bool {
	return game.actions.validate()
}

func (game *Game) removePlayer(name string) {
	if player, exists := game.players[name]; exists {
		game.actions[player.team][player.role] -= 1
		game.playerActions[player.team][player.role] -= 1
		delete(game.players, name)
	}
}

func (game *Game) removeGame(message any) bool {
	return game.manager.removeGame(game.name, message)
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

func getPlayerActions(players ClientList) Actions {
	actions := Actions{
		red: {
			cluegiver: 0,
			guesser: 0,
		},
		blue: {
			cluegiver: 0,
			guesser: 0,
		},
	}

	for _, player := range players {
		actions[player.team][player.role] += 1
	}
	return actions
}

func getActions(players ClientList, bots *BotActions) (Actions, Actions) {
	allActions := Actions{
		red: {
			cluegiver: 0,
			guesser: 0,
		},
		blue: {
			cluegiver: 0,
			guesser: 0,
		},
	}
	// Go doesn't have a "deep copy" function
	playerActions := getPlayerActions(players)

	// add bot actions to player actions
	for _, t := range []Team{ red, blue } {
		for _, r := range []Role{ guesser, cluegiver } {
			allActions[t][r] = playerActions[t][r]
			if bots != nil && bots.hasTeamAction(t, r) {
				allActions[t][r] += 1
			}
		}
	}
	return playerActions, allActions
}