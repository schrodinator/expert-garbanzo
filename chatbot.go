package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
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
	OpenAI     *openai.Client
	game       *Game
	clue_chan  chan *ClueStruct
	guess_chan chan *ClueStruct
	actions    *BotActions
	client     *Client
}

type BotActions struct {
	Guesser   TeamActions `json:"guesser"`
	Cluegiver TeamActions `json:"cluegiver"`
}
type TeamActions struct {
	Red  bool `json:"red"`
	Blue bool `json:"blue"`
}
func (ba BotActions) hasAction(r Role) bool {
	switch r {
	case cluegiver:
		return ba.Cluegiver.Red || ba.Cluegiver.Blue
	case guesser:
		return ba.Guesser.Red || ba.Guesser.Blue
	default:
		return false
	}
}
func (ba BotActions) hasTeamAction(t Team, r Role) bool {
	var ta TeamActions
	switch r {
	case cluegiver:
		ta = ba.Cluegiver
		break;
	case guesser:
		ta = ba.Guesser
		break;
	default:
		return false
	}
	switch t {
	case red:
		return ta.Red
	case blue:
		return ta.Blue
	default:
		return false
	}
}

func NewBot(game *Game, ba *BotActions) *Bot {
	b := &Bot{
		ctx:     context.TODO(),
		OpenAI:  openai.NewClient(token),
		game:    game,
		actions: ba,
		client:  &Client{game: game,},
	}

	if ba.hasAction(cluegiver) {
		b.clue_chan = b.makeClue()
	}
	if ba.hasAction(guesser) {
		b.guess_chan = b.makeGuess()
	}

	return b
}

func (bot *Bot) Play(clue GiveClueEvent) (string, *ClueStruct) {
	game := bot.game
	t := game.teamTurn
	r := game.roleTurn
	if game.score[t] <= 0 {
		// No cards left to guess.
		return "", nil
	}
	if bot.actions.hasTeamAction(t, r) {
		game.notifyPlayers(EventBotWait, nil)
		bot.client.team = t
		bot.client.role = r
		c := &ClueStruct{
			capsWords: make([]string, 0),
		}
		var e string
		switch r {
		case cluegiver:
			e = EventGiveClue
			c.word = t.String()
			bot.clue_chan <- c
			c =<-bot.clue_chan
			break;
		case guesser:
			e = EventMakeGuess
			c.word = clue.Clue
			if clue.NumCards > 0 {
				c.numGuess = clue.NumCards
			} else {
				/* Unspecified number of cards,
				   unlimited guesses. But actually
				   limit it to the number of cards
				   remaining for this team. */
				c.numGuess = game.score[game.teamTurn]
			}
			bot.guess_chan <- c
			c =<-bot.guess_chan
			break;
		}
		return e, c
	}
	return "", nil
}

/* Real call to OpenAI ChatGPT. Function stored in a var so it
   can be overridden for testing. */
var askGPT3Dot5Bot = func (bot *Bot, system string, user string) (openai.ChatCompletionResponse, error) {
	return bot.OpenAI.CreateChatCompletion(
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
			"it may not be a plural of any of the words). " +
			"A good clue is a synonym or a related word that evokes as " +
			"many words on your team's word list as possible, without " +
			"evoking any of the opposing team's words. When prompted, " +
			"reply with ONLY the following: " +
			"your clue, the number of words from your team's list " +
			"that match your clue, and the specific words " +
			"that match your clue."

		for {
			var clue *ClueStruct
			var ok bool
			select {
			case clue, ok = <-c:
				if !ok {
					clue.err = fmt.Errorf("Clue channel error")
					break
				}

				w := bot.game.cards.getClueWords(bot.game.teamTurn)
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
				if verbose {
					fmt.Printf(clue.response)
				}
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
	clue.capsWords, _ = findUniqueAllCapsWords(respStr)
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
	if len(words) > 1 && strings.HasPrefix(words[0], "Clue") {
		// e.g. "Clue: ..."
		word = words[1]
	} else if len(words) > 2 && strings.Compare(words[1], "clue") == 0 {
		if len(words) == 3 {
			// e.g. "My clue: ..." or "The clue: ..."
			word = words[2]
		} else {
			// e.g. "My clue is ..." or "The clue is: ..."
			word = words[3]
		}
	}
	return strings.Trim(word, "\".")
}

func parseGPTResponseMatches(respStr string) string {
	/* Words are upper case and at least 2 letters long.
       Words could be separated by a comma and/or a space. */
	re := regexp.MustCompile("([[:upper:]]{2,}[, ]{1,2})*[[:upper:]]{2,}")
	return re.FindString(respStr)
}

func findGuessWords(respStr string) ([]string, error) {
	/* ChatBot usually states their guesses as all-caps words. */
	s, err := findUniqueAllCapsWords(respStr)
	if len(s) > 0 {
		return s, err
	}
	/* Possibility of not-all-caps words in numbered list. */
	s, err = findUniqueNumberedListWords(respStr)
	if len(s) > 0 {
		return s, err
	}
	/* Possibility of not-all-caps words in quotation marks. */
	s, err = findUniqueWordsInQuotes(respStr)
	if len(s) > 0 {
		return s, err
	}
	return s, fmt.Errorf("could not find any guess words")
}

func findUniqueAllCapsWords(respStr string) ([]string, error) {
	re := regexp.MustCompile("[[:upper:]]{2,}")
	match := re.FindAllString(respStr, -1)
	if len(match) == 0 {
		return match, fmt.Errorf("could not find any all-caps words")
	}
	return unique(match), nil
}

func findUniqueNumberedListWords(respStr string) ([]string, error) {
	re := regexp.MustCompile("[1-9][.)]? \"?([[:alpha:]]{2,})")
	match := re.FindAllStringSubmatch(respStr, -1)
	if len(match) == 0 {
		return []string{}, fmt.Errorf("could not find a numbered list of words")
	}
	s := make([]string, len(match))
	for i := 0; i < len(match); i++ {
		/* match is a 2D slice formatted like: 
			[[fullmatch0 capturegroup0] [fullmatch1 capturegroup1] ...] */
		s[i] = strings.ToUpper(match[i][1])
	}
	return unique(s), nil
}

func findUniqueWordsInQuotes(respStr string) ([]string, error) {
	re := regexp.MustCompile("\"[[:alpha:]]{2,}[.]?\"")
	match := re.FindAllString(respStr, -1)
	if len(match) == 0 {
		return match, fmt.Errorf("could not find any words in quotation marks")
	}
	for i := 0; i < len(match); i++ {
		match[i] = strings.ToUpper(strings.Trim(match[i], ".\""))
	}
	return unique(match), nil
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
	return uniqueSlice
}

func (bot *Bot) makeGuess() chan *ClueStruct {
	c := make(chan *ClueStruct)

	go func(bot *Bot) () {
		prompt := "You are playing a word game. Your teammate " +
		"will give you a clue and a number. " +
		"Choose that number of words from your word list that " +
		"best match the clue. Reply with only those words, in " +
		"ALL CAPITAL LETTERS."

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

				message := fmt.Sprintf(
					"The word list is: %s. The clue is: %s. The number is: %d",
					words, clue.word, clue.numGuess)

				resp, err := bot.askGPT3Dot5(prompt, message)
				if err != nil {
					clue.err = fmt.Errorf("ChatCompletion error: %v", err)
					break
				}
				clue.response = resp.Choices[0].Message.Content
				clue.capsWords, clue.err = findGuessWords(clue.response)
			}
			if verbose {
				fmt.Printf(clue.response)
			}
			c <- clue
		}
	}(bot)

	return c
}