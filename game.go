package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
)

var (
	dictionary []string
	dictLen    int
)

const numCards = 25

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

func getGameWords() map[string]string {
	cards := make(map[string]string, numCards)
	var alignments = [25]string{
		"red", "red", "red", "red", "red", "red", "red", "red", "red",
	    "blue", "blue", "blue", "blue", "blue", "blue", "blue", "blue",
	    "assassin",
	    "neutral", "neutral", "neutral", "neutral", "neutral", "neutral", "neutral"}
	rand.Shuffle(numCards, func(i, j int) {
		alignments[i], alignments[j] = alignments[j], alignments[i]
	})
	i := 0
	for i < numCards {
		word := dictionary[rand.Intn(dictLen)]
		// ensure each word is unique
		if _, exists := cards[word]; exists {
			continue
		}
		cards[word] = alignments[i]
		i++
	}
	return cards
}