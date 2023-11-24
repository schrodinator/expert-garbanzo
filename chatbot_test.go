package main

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
)

/* Check if the error message in "out" contains the
   error message in "want" */
   func ErrorContains(t *testing.T, out error, want error) bool {
	t.Helper()

    if out == nil {
        return want == nil
    }
    if want == nil {
        return false
    }
    return strings.Contains(out.Error(), want.Error())
}

func getSomeCards() *Game {
	game := &Game{}
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
	return game
}

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
	match := parseGPTResponseMatches(respStr)
	if match != "SCALE, WATCH, MAPLE" {
		t.Errorf("Expected SCALE, WATCH, MAPLE. Got %v", match)
	}

	respStr = "IT MATCHES, THE, FIRST, GROUP, OF ALL CAPS WORDS, "+ 
		"possibly separated by commas, " +
		"BUT NOT THE LAST COMMA IN THE SET"
	match = parseGPTResponseMatches(respStr)
	if match != "IT MATCHES, THE, FIRST, GROUP, OF ALL CAPS WORDS" {
		t.Errorf("Expected IT MATCHES, THE, FIRST, GROUP, OF ALL CAPS WORDS. Got %v", match)
	}

	respStr = "There are no all-caps words here."
	match = parseGPTResponseMatches(respStr)
	if match != "" {
		t.Errorf("Expected empty string, got %v", match)
	}
}

func TestUnique(t *testing.T) {
	s := []string{"These", "some", "are", "always", "some", "words", "words", "words"}
	u := unique(s)
	if slices.Compare(u, []string{"These", "always", "are", "some", "words"}) != 0 {
		t.Errorf("got: %v", u)
	}
}

func TestFindAllCapsWords(t *testing.T) {
	s := "THIS is a SENTENCE with SOME CAPS in it."
	words := findUniqueAllCapsWords(s)
	if slices.Compare(words, []string{"CAPS", "SENTENCE", "SOME", "THIS"}) != 0 {
		t.Errorf("got: %v", words)
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
	if c.response != respStr {
		t.Errorf("response: expected response string, got %v", c.response)
	}
	if c.match != "SCALE, WATCH, MAPLE" {
		t.Errorf("match: expected SCALE, WATCH, MAPLE, got %v", c.match)
	}
	if slices.Compare(c.capsWords, []string{"MAPLE", "SCALE", "WATCH"}) != 0 {
		t.Errorf("capsWords: expected MAPLE, SCALE, WATCH, got %v", c.capsWords)
	}
	if c.err != nil {
		t.Errorf("got non-nil err: %v", c.err)
	}
}

func TestMakeClueReal(t *testing.T) {
	token = getMasterPassword("gpt-secretkey.txt")
	game := getSomeCards()
	bot := NewBot(game)
	clue := &ClueStruct{
		word: "red",
		capsWords: make([]string, 3),	
	}	
	bot.clue_chan <-clue
	clue = <-bot.clue_chan
	fmt.Printf("%#v", clue)

	// Weak tests for a nondeterministic response
	if (clue.numGuess > 9 || clue.numGuess < 1) {
		t.Errorf("numGuess out of range: %v", clue.numGuess)
	}
	if clue.word == "" {
		t.Errorf("clue word not present")
	}
	if len(clue.capsWords) == 0 {
		t.Errorf("caps words not present")
	}
}

func TestMakeClueMock(t *testing.T) {
	game := getSomeCards()
	bot := NewBot(game)

	type testStruct struct {
		name        string
		botResp     string
		expectNum   int
		expectWord  string
		expectMatch string
		expectErr   error
	}
	tests := []testStruct{
		{
			name: "A straightforward response",
			botResp: "Clue: Measure\nNumber of words that " +
				"match the clue: 3\nWords that match the clue: " +
				"SCALE, WATCH, MAPLE",
			expectNum: 3,
			expectWord: "Measure",
			expectMatch: "SCALE, WATCH, MAPLE",
			expectErr: nil,
		},
		{
			name: "Response with no number",
			botResp: "Clue: My\nWords that evoke the clue: " +
				"SHADOW SHOW SLIP VACUUM WATCH WITCH",
			expectNum: 6,
			expectWord: "My",
			expectMatch: "SHADOW SHOW SLIP VACUUM WATCH WITCH",
			expectErr: fmt.Errorf("Could not parse number in ChatCompletion response"),
		},
		{
			/* If we can't find all-caps words in the input string,
			   the log should be the entire input string */
			name: "Response with no matched explanation words",
			botResp: "Encoding 2",
			expectNum: 2,
			expectWord: "Encoding",
			expectMatch: "",
			expectErr: nil,
		},
	}

	for _, test := range tests {
		askGPT3Dot5Bot = func (bot *Bot, system string, user string) (openai.ChatCompletionResponse, error) {
			return openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Content: test.botResp,
						},
					},
				},
			}, nil
		}
		clue := &ClueStruct{
			word: "red",
			capsWords: make([]string, 3),
		}
		bot.clue_chan <-clue
		clue = <-bot.clue_chan

		if !ErrorContains(t, clue.err, test.expectErr) {
			t.Errorf("%v Error: expected \"%v\", got \"%v\"", test.name, test.expectErr, clue.err)
		}
		if clue.numGuess != test.expectNum {
			t.Errorf("%v numGuess: expected %v, got %v", test.name, test.expectNum, clue.numGuess)
		}
		if clue.word != test.expectWord {
			t.Errorf("%v clue word: expected %v, got %v", test.name, test.expectWord, clue.word)
		}
		if clue.response != test.botResp {
			t.Errorf("%v clue response: expected %v, got %v", test.name, test.botResp, clue.response)
		}
		if clue.match != test.expectMatch {
			t.Errorf("%v clue log[1]: expected %v, got %v", test.name, test.expectMatch, clue.match)
		}
	}
}

func TestMakeGuessReal(t *testing.T) {
	token = getMasterPassword("gpt-secretkey.txt")
	game := getSomeCards()
	bot := NewBot(game)
	clue := &ClueStruct{
		word: "Measure",
		numGuess: 4,
	}	
	bot.guess_chan <-clue
	clue = <-bot.guess_chan
	fmt.Printf("%#v", clue)
}