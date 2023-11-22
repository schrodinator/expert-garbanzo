package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type ClueStruct struct {
	numGuess int
	word     string
	log      string
}

type Bot struct {
	ctx       context.Context
	client    *openai.Client
	game      *Game
	clue_in   chan string
	clue_out  chan ClueStruct
	guess_in  chan ClueStruct
	guess_out chan string
}

func NewBot(game *Game) *Bot {
	return &Bot{
		ctx:       context.TODO(),
		client:    openai.NewClient(token),
		game:      game,
	}
}

func ClueBot(game *Game) *Bot {
	b := &Bot{
		ctx:       context.TODO(),
		client:    openai.NewClient(token),
		game:      game,
		clue_in:   make(chan string),
		clue_out:  make(chan ClueStruct),
	}

	go b.makeClue()

	return b
}

func GuessBot(game *Game) *Bot {
	b := &Bot{
		ctx:       context.TODO(),
		client:    openai.NewClient(token),
		game:      game,
		guess_in:  make(chan ClueStruct),
		guess_out: make(chan string),
	}

	go b.makeGuess()

	return b
}

func (bot *Bot) askGPT3Dot5(system string, user string) (openai.ChatCompletionResponse, error) {
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

func (bot *Bot) makeClue() {
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
		"state ONLY the following: " +
		"your clue, the number of words from your team's list " +
		"that match your clue, and the specific words " +
		"that match your clue."

	for {
		select {
		case team, ok := <-bot.clue_in:
			if !ok {
				log.Println("Clue egress error")
				return
			}

			w := bot.game.cards.getClueWords(team)
			if len(w.myTeam) == 0 || len(w.others) == 0 {
				log.Println("makeClue error: got zero-length word list")
				return
			}

			message := fmt.Sprintf("Your team's list: %s. Opposing team's list: %s.",
				w.myTeam, w.others)

			resp, err := bot.askGPT3Dot5(prompt, message)
			if err != nil {
				log.Printf("ChatCompletion error: %v", err)
				return
			}

			respStr := resp.Choices[0].Message.Content

			var clue ClueStruct
			err = parseGPTResponse(respStr, &clue)
			if err != nil {
				log.Println(err)
				return
			}

			bot.clue_out <- clue

		}
	}
}

func parseGPTResponse(respStr string, clue *ClueStruct) error {
	/* Chat Bot 3.5 replies in an inconsistent format, despite
	   my attempts at prompt engineering. */
	i, err := parseGPTResponseNumber(respStr)
	if err != nil {
		return err
	}
	word := parseGPTResponseClue(respStr)
	remaining := parseGPTResponseMatches(respStr, i)

	clue.numGuess = i
	clue.word = word
	clue.log = remaining

	return nil
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

func parseGPTResponseMatches(respStr string, num int) string {
	/* If we can't find "num" matching words,
	   return the original response string. */
	remaining := respStr
	s := make([]string, num)
	for i := 0; i < num; i++ {
		// Words are uppercase and at least 2 letters long
		s[i] = "[[:upper:]]{2,}"
	}
	// Words could be separated by a comma and/or a space
	multi := strings.Join(s, "[, ]{1,2}")
	re := regexp.MustCompile(multi)
	matches := re.FindString(respStr)
	if matches != "" {
		remaining = matches
	}
	return remaining
}

func (bot *Bot) makeGuess() {
	prompt := "You are playing a word game. Your teammate " +
	"will give you a clue and a number. " +
	"Choose that number of words from your word list that " +
	"best match the clue."

	for {
		select {
		case clue, ok := <-bot.guess_in:
			if !ok {
				log.Println("Guess egress error")
				return
			}

			words := bot.game.cards.getGuessWords()
			if len(words) == 0 {
				log.Println("makeGuess error: got zero-length word list")
				return
			}

			// TODO: handle case of infinite guesses / unspecified number of cards
			message := fmt.Sprintf(
				"The word list is: %s. The clue is: %s. The number is: %d",
				words, clue.word, clue.numGuess-1)

			resp, err := bot.askGPT3Dot5(prompt, message)
			if err != nil {
				log.Printf("ChatCompletion error: %v", err)
				return
			}

			bot.guess_out <- resp.Choices[0].Message.Content
		}
	}
}
