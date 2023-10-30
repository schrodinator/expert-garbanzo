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
	i := 0
	for i < numCards {
		word := dictionary[rand.Intn(dictLen)]
		// ensure each word is unique
		if _, exists := cards[word]; exists {
			continue
		}
		cards[word] = "white"
		i++
	}
	return cards
}

func getAlignments(cards map[string]string) {
	var alignments = [25]string{
		"red", "red", "red", "red", "red", "red", "red", "red", "red",
	    "blue", "blue", "blue", "blue", "blue", "blue", "blue", "blue",
	    "assassin",
	    "neutral", "neutral", "neutral", "neutral", "neutral", "neutral", "neutral"}
	i := 0
	/* iteration over the map happens in random order,
	   so we are effectively randomizing the alignments */
	for word := range cards {
		cards[word] = alignments[i]
		i++
	}
}