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

func getSomeCards(t *testing.T, ba *BotActions) *Game {
	t.Helper()
	
	manager := setupGame(t, nil, ba)
	game := manager.games["test"]
	game.players = nil
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

	respStr = "K9 10"
	num, err = parseGPTResponseNumber(respStr)
	if err != nil {
		t.Error(err)
	}
	if num != 10 {
		t.Errorf("Got %v, expected 10", num)
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

	respStr = "I will give the clue \"bobcat\", There is 1 word that matches the clue: KITTY."
	word = parseGPTResponseClue(respStr)
	if word != "bobcat" {
		t.Errorf("Got %v, expected bobcat", word)
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
	if slices.Compare(u, []string{"These", "some", "are", "always", "words"}) != 0 {
		t.Errorf("got: %v", u)
	}
}

func TestFindAllCapsWords(t *testing.T) {
	s := "THIS is a SENTENCE with SOME CAPS in it."
	words, err := findUniqueAllCapsWords(s)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Compare(words, []string{"THIS", "SENTENCE", "SOME", "CAPS"}) != 0 {
		t.Errorf("got: %v", words)
	}
}

func TestFindUniqueNumberedListWords(t *testing.T) {
	s := "Here are my guesses:\n1. First\n2. Second\n3. Third\n4. Third"
	words, err := findUniqueNumberedListWords(s)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Compare(words, []string{"FIRST", "SECOND", "THIRD"}) != 0 {
		t.Errorf("got: %v", words)
	}

	s = "There is no list\nin this string"
	words, err = findUniqueNumberedListWords(s)
	if err == nil || len(words) > 0 {
		t.Errorf("expected no match, got %v", words)
	}
}

func TestFindUniqueWordsInQuotes(t *testing.T) {
	s := "My guesses are \"first\" and \"second.\""
	words, err := findUniqueWordsInQuotes(s)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Compare(words, []string{"FIRST", "SECOND"}) != 0 {
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
	if slices.Compare(c.capsWords, []string{"SCALE", "WATCH", "MAPLE"}) != 0 {
		t.Errorf("capsWords: expected SCALE, WATCH, MAPLE, got %v", c.capsWords)
	}
	if c.err != nil {
		t.Errorf("got non-nil err: %v", c.err)
	}
}

func TestMakeClueReal(t *testing.T) {
	if testing.Short() {
		t.Skip("Real ChatGPT test skipped in short mode")
	}

	token = getGPTToken("gpt-secretkey.txt")
	ba := &BotActions{
		Cluegiver: TeamActions{
			Red: true,
		},
	}
	game := getSomeCards(t, ba)
	game.roleTurn = cluegiver
	bot := game.bot
	clue := &ClueStruct{
		word: "red",
		capsWords: make([]string, 0),	
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
	ba := &BotActions{
		Cluegiver: TeamActions{
			Red: true,
		},
	}
	game := getSomeCards(t, ba)
	game.roleTurn = cluegiver
	bot := game.bot

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
			expectErr: fmt.Errorf("could not parse number in ChatCompletion response"),
		},
		{
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
			t.Errorf("%v clue match: expected %v, got %v", test.name, test.expectMatch, clue.match)
		}
	}
}

func TestMakeGuessReal(t *testing.T) {
	if testing.Short() {
		t.Skip("Real ChatGPT test skipped in short mode")
	}
	
	token = getGPTToken("gpt-secretkey.txt")
	ba := &BotActions{
		Guesser: TeamActions{
			Red: true,
		},
	}
	game := getSomeCards(t, ba)
	game.roleTurn = guesser
	bot := game.bot
	clue := &ClueStruct{
		word: "Measure",
		numGuess: 4,
		capsWords: make([]string, 0),
	}
	bot.guess_chan <-clue
	clue = <-bot.guess_chan
	fmt.Printf("%#v", clue)

	// Weak tests for a nondeterministic response
	if clue.response == "" {
		t.Errorf("clue response not present")
	}
	if len(clue.capsWords) == 0 {
		t.Errorf("caps words not present")
	}
}

func TestMakeGuessMock(t *testing.T) {
	ba := &BotActions{
		Guesser: TeamActions{
			Red: true,
		},
	}
	game := getSomeCards(t, ba)
	bot := game.bot
	clue := &ClueStruct{
		word: "Measure",
		numGuess: 4,
	}

	response := "The three words from the word list that best match " +
			    "the clue \"Measure\" are:\n1. SCALE\n2. RULER\n3. TAPE"
	askGPT3Dot5Bot = func (bot *Bot, system string, user string) (openai.ChatCompletionResponse, error) {
		return openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Content: response,
					},
				},
			},
		}, nil
	}

	bot.guess_chan <-clue
	clue = <-bot.guess_chan

	if clue.response != response {
		t.Errorf("got response: %v", clue.response)
	}
	if clue.err != nil {
		t.Errorf("got err: %v", clue.err)
	}
	if slices.Compare(clue.capsWords, []string{"SCALE", "RULER", "TAPE"}) != 0 {
		t.Errorf("got caps words: %v", clue.capsWords)
	}
}

func TestBotPlayCluegiver(t *testing.T) {
	ba := &BotActions{
		Cluegiver: TeamActions{
			Red: true,
		},
	}
	game := getSomeCards(t, ba)
	bot := game.bot

	response := "Clue: Measure\nNumber of words that " +
		"match the clue: 3\nWords that match the clue: " +
		"SCALE, WATCH, MAPLE"
	askGPT3Dot5Bot = func (bot *Bot, system string, user string) (openai.ChatCompletionResponse, error) {
		return openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Content: response,
					},
				},
			},
		}, nil
	}

	e, c, _, _ := bot.Play(GiveClueEvent{})
	if e == "" || c == nil {
		t.Fatal("Play returned empty values")
	}
	if e != EventGiveClue {
		t.Errorf("Event: expected %v, got %v", EventGiveClue, e)
	}
	if c.numGuess != 3 {
		t.Errorf("numGuess: expected 3, got %v", c.numGuess)
	}
	if c.word != "Measure" {
		t.Errorf("clue word: expected 'Measure', got '%v'", c.word)
	}
	if c.response != response {
		t.Errorf("clue response: expected %v, got %v", response, c.response)
	}
}

func TestBotPlayGuesser(t *testing.T) {
	ba := &BotActions{
		Cluegiver: TeamActions{
			Red: true,
		},
	}
	game := getSomeCards(t, ba)
	bot := game.bot

	clue := GiveClueEvent {
		Clue: "Measure",
		NumCards: 3,
	}
	response := "The three words from the word list that best match " +
			    "the clue \"Measure\" are:\n1. SCALE\n2. RULER\n3. TAPE"
	askGPT3Dot5Bot = func (bot *Bot, system string, user string) (openai.ChatCompletionResponse, error) {
		return openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Content: response,
					},
				},
			},
		}, nil
	}

	e, c, _, _ := bot.Play(clue)
	if e == "" || c == nil {
		t.Fatal("Play returned empty values")
	}
	if c.response != response {
		t.Errorf("got response: %v", c.response)
	}
	if slices.Compare(c.capsWords, []string{"SCALE", "RULER", "TAPE"}) != 0 {
		t.Errorf("got caps words: %v", c.capsWords)
	}
}