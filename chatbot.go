package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/exp/slices"
)

type ClueStruct struct {
	response  string
	numGuess  int
	word      string
	match     string
	capsWords []string
	err       error
}

type Bot struct {
	ctx        context.Context
	client     *openai.Client
	game       *Game
	clue_chan  chan *ClueStruct
	guess_chan chan *ClueStruct
}

func NewBot(game *Game) *Bot {
	b := &Bot{
		ctx:       context.TODO(),
		client:    openai.NewClient(token),
		game:      game,
	}

	b.clue_chan = b.makeClue()
	b.guess_chan = b.makeGuess()

	return b
}

var askGPT3Dot5Bot = func (bot *Bot, system string, user string) (openai.ChatCompletionResponse, error) {
	return bot.client.CreateChatCompletion(
		bot.ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: system,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: user,
				},
			},
		},
	)
}

func (bot *Bot) askGPT3Dot5(system string, user string) (openai.ChatCompletionResponse, error) {
	return askGPT3Dot5Bot(bot, system, user)
}

func (bot *Bot) makeClue() chan *ClueStruct {
	c := make(chan *ClueStruct)

	go func(bot *Bot) () {
		prompt := "You are playing a word game. In this game, " +
			"your objective is to give a one-word clue that will " +
			"help your team guess as many words as possible " +
			"from your team's word list, while NOT guessing words " +
			"from the opposing team's word list. The clue must not " +
			"exactly match any of the words on the lists, nor may it " +
			"be a direct derivative of any of the words (for example, " +
			"it may not be a plural of any of the words). Instead, a " +
			"good clue is a synonym or a related word that evokes as " +
			"many words on your team's word list as possible, without " +
			"also evoking the opposing team's words. When prompted, " +
			"state the following: " +
			"your clue, the number of words from your team's list " +
			"that match your clue, and the specific words " +
			"that match your clue. Explain your reasoning."

		for {
			var clue *ClueStruct
			var ok bool
			select {
			case clue, ok = <-c:
				if !ok {
					clue.err = fmt.Errorf("Clue channel error")
					break
				}

				w := bot.game.cards.getClueWords(clue.word)
				if len(w.myTeam) == 0 || len(w.others) == 0 {
					clue.err = fmt.Errorf("makeClue error: got zero-length word list")
					break
				}

				message := fmt.Sprintf("Your team's list: %s. Opposing team's list: %s.",
					w.myTeam, w.others)

				resp, err := bot.askGPT3Dot5(prompt, message)
				if err != nil {
					clue.err = fmt.Errorf("ChatCompletion error: %v", err)
					break
				}

				respStr := resp.Choices[0].Message.Content

				parseGPTResponse(respStr, clue)
			}
			c <- clue
		}
	}(bot)

	return c
}

func parseGPTResponse(respStr string, clue *ClueStruct) {
	/* Chat Bot 3.5 replies in an inconsistent format, despite
	   my attempts at prompt engineering. */
	clue.response = respStr
	clue.word = parseGPTResponseClue(respStr)
	clue.match = parseGPTResponseMatches(respStr)
	clue.capsWords = findUniqueAllCapsWords(respStr)
	var i int
	i, clue.err = parseGPTResponseNumber(respStr)
	/* If we didn't find a number but did find all-caps words,
	   use the number of all-caps words. */
	if clue.err != nil && len(clue.capsWords) != 0 {
		/* The words might be separated by spaces and/or commas */
		i = len(clue.capsWords)
	}
	clue.numGuess = i
}

func parseGPTResponseNumber(respStr string) (int, error) {
	numRegex := regexp.MustCompile("[0-9]+")
	nums := numRegex.FindAllString(respStr, 1)
	if nums == nil {
		return -1, fmt.Errorf("Could not parse number in ChatCompletion response: %v", respStr)
	}
	// string to int
	i, err := strconv.Atoi(nums[0])
	if err != nil {
		return -1, fmt.Errorf("Could not convert number in ChatCompletion response: %v, %v",
			nums, respStr)
	}
	return i, nil
}

func parseGPTResponseClue(respStr string) string {
	line1, _, _ := strings.Cut(respStr, "\n")
	words := strings.Split(line1, " ")
	word := words[0]
	if strings.HasPrefix(words[0], "Clue") && len(words) > 1 {
		word = words[1]
	}
	return word
}

func parseGPTResponseMatches(respStr string) string {
	/* Words are upper case and at least 2 letters long.
       Words could be separated by a comma and/or a space. */
	re := regexp.MustCompile("([[:upper:]]{2,}[, ]{1,2})*[[:upper:]]{2,}")
	return re.FindString(respStr)
}

func findUniqueAllCapsWords(respStr string) []string {
	re := regexp.MustCompile("[[:upper:]]{2,}")
	match := re.FindAllString(respStr, -1)
	return unique(match)
}

func unique(slice []string) []string {
    seen := make(map[string]bool)
    uniqueSlice := []string{}
    for _, v := range slice {
        if !seen[v] {
            seen[v] = true
            uniqueSlice = append(uniqueSlice, v)
        }
    }
    slices.Sort(uniqueSlice)
	return uniqueSlice
}

func (bot *Bot) makeGuess() chan *ClueStruct {
	c := make(chan *ClueStruct)

	go func(bot *Bot) () {
		prompt := "You are playing a word game. Your teammate " +
		"will give you a clue and a number. " +
		"Choose that number of words from your word list that " +
		"best match the clue."

		for {
			var clue *ClueStruct
			var ok bool
			select {
			case clue, ok = <-c:
				if !ok {
					clue.err = fmt.Errorf("Guess channel error")
					break
				}

				words := bot.game.cards.getGuessWords()
				if len(words) == 0 {
					clue.err = fmt.Errorf("makeGuess error: got zero-length word list")
					break
				}

				// TODO: handle case of infinite guesses / unspecified number of cards
				message := fmt.Sprintf(
					"The word list is: %s. The clue is: %s. The number is: %d",
					words, clue.word, clue.numGuess-1)

				resp, err := bot.askGPT3Dot5(prompt, message)
				if err != nil {
					clue.err = fmt.Errorf("ChatCompletion error: %v", err)
					break
				}
				clue.response = resp.Choices[0].Message.Content

			}
			c <- clue
		}
	}(bot)

	return c
}
