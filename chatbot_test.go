package main

import (
	"fmt"
	"testing"
)

func TestParseGPTResponseNumber(t *testing.T) {
	respStr := "Clue: Based\nMatches: 3\nTo kids these days, " +
	"it means expressing a strong opinion"
	num, err := parseGPTResponseNumber(respStr)
	if err != nil {
		t.Error(err)
	}
	if num != 3 {
		t.Errorf("Got %v, expected 3", num)
	}

	respStr = "It will 4 only 3 return 2 the 1st number"
	num, err = parseGPTResponseNumber(respStr)
	if err != nil {
		t.Error(err)
	}
	if num != 4 {
		t.Errorf("Got %v, expected 4", num)
	}

	respStr = "There aren't any numbers here"
	num, err = parseGPTResponseNumber(respStr)
	if err == nil {
		t.Error("Expected an error")
	}
	if num != -1 {
		t.Errorf("Got %v, expected -1", num)
	}
}

func TestParseGPTResponseClue(t *testing.T) {
	respStr := "Clue: Based\nMatches: 3\nTo kids these days, " +
	"it means expressing a strong opinion"
	word := parseGPTResponseClue(respStr)
	if word != "Based" {
		t.Errorf("Got %v, expected Based", word)
	}

	respStr = "Returns the first word if it doesn't match Clue"
	word = parseGPTResponseClue(respStr)
	if word != "Returns" {
		t.Errorf("Got %v, expected Returns", word)
	}
}

func TestParseGPTResponseMatches(t *testing.T) {
	respStr := "Clue: Measure\nNumber of words that " +
	"match the clue: 3\nWords that match the clue: " +
	"SCALE, WATCH, MAPLE"
	match := parseGPTResponseMatches(respStr, 3)
	if match != "SCALE, WATCH, MAPLE" {
		t.Errorf("Expected SCALE, WATCH, MAPLE. Got %v", match)
	}

	respStr = "If it can't match # NUM OF WORDS, it " +
	"returns the input string"
	match = parseGPTResponseMatches(respStr, 4)
	if match != respStr {
		t.Errorf("Expected %v. Got %v", respStr, match)
	}

	respStr = "IT MATCHES THE FIRST NUM ALL CAPS WORDS"
	match = parseGPTResponseMatches(respStr, 2)
	if match != "IT MATCHES" {
		t.Errorf("Expected IT MATCHES. Got %v", match)
	}
}

func TestParseGPTResponse(t *testing.T) {
	respStr := "Clue: Measure\nNumber of words that " +
	"match the clue: 3\nWords that match the clue: " +
	"SCALE, WATCH, MAPLE"
	var c ClueStruct
	parseGPTResponse(respStr, &c)
	if c.numGuess != 3 {
		t.Errorf("numGuess: expected 3, got %v", c.numGuess)
	}
	if c.word != "Measure" {
		t.Errorf("word: expected Measure, got %v", c.word)
	}
	if c.log != "SCALE, WATCH, MAPLE" {
		t.Errorf("log: expected SCALE, WATCH, MAPLE, got %v", c.log)
	}
}

func TestMakeClue(t *testing.T) {
	manager := setupGame(t, nil)
	game := manager.games["test"]
	game.cards = Deck{
		"AMAZON": "blue",
		"BOOT": "blue",
		"BOX": "blue",
		"CLUB": "neutral",
		"FILE": "red",
		"HORSE": "red",
		"ICE": "red",
		"LOG": "neutral",
		"MAPLE": "red",
		"MOUSE": "red",
		"NEEDLE": "blue",
		"OIL": "neutral",
		"OLIVE": "black",
		"PILOT": "neutral",
		"POINT": "blue",
		"ROCKET": "blue",
		"SCALE": "red",
		"SHADOW": "neutral",
		"SHOE": "neutral",
		"SLIP": "blue",
		"SPIDER": "blue",
		"STAR": "neutral",
		"TAP": "red",
		"VACUUM": "red",
		"WATCH":"red",
	}
	token = getMasterPassword("gpt-secretkey.txt")
	bot := ClueBot(game)	
	bot.clue_in <-"red"
	clue := <-bot.clue_out
	fmt.Printf("%#v", clue)
	close(bot.clue_in)
	close(bot.clue_out)

	// Weak tests for a nondeterministic response
	if (clue.numGuess > 9 || clue.numGuess < 1) {
		t.Errorf("numGuess out of range: %v", clue.numGuess)
	}
	if clue.word == "" {
		t.Errorf("clue word not present")
	}
	if clue.log == "" {
		t.Errorf("clue log not present")
	}
}