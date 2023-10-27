package main

import (
	"fmt"
	"io"
	"os"
)

var (
	masterPassword = getMasterPassword()
	defaultChatroom = "general"
)

func getMasterPassword() string {
	file, err := os.Open("password.txt")
	if err != nil {
        fmt.Println("Error opening password.txt:", err)
        return ""
    }
    defer file.Close() // Close the file when we're done

	pword, err := io.ReadAll(file)
    if err != nil {
        fmt.Println("Error reading password.txt:", err)
        return ""
    }
	return string(pword)
}