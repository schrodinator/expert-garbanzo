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

func getGameWords() []string {
	numWords := 25

	words := make([]string, numWords)
	for i := 0; i < numWords; i++ {
		words[i] = dictionary[rand.Intn(dictLen)]
	}
	return words
}