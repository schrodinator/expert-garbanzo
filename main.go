package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

var masterPassword string
var token string

/* Print ChatGPT responses to stdout. */
var verbose bool

const (
	defaultChatRoom = "lobby"
	deathCard       = "black"
)

func main() {
	verbose = true
	masterPassword = getMasterPassword("password.txt")
	token = getMasterPassword("gpt-secretkey.txt")

	setupAPI()

	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
}

// At the point where you start running multiple instances, it is common to include
// Redis or RabbitMQ to allow distributed messages for the websockets.
// You would then listen on the PubSub schema used and push messages on RabbitMQ/Redis,
// then read from those topics and push onto the Websockets.
func setupAPI() {
	ctx := context.Background()

	readDictionary("./codenames-wordlist.txt")

	manager := NewManager(ctx)

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.serveWS)
	http.HandleFunc("/login", manager.loginHandler)
}

func getMasterPassword(path string) string {
	file, err := os.Open(path)
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
