package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
)

var (
	token string
	verbose bool
)

const (
	defaultChatRoom = "lobby"
	deathCard       = "black"
)

func main() {
	/* Log ChatGPT responses */
	verbose = true

	token = getGPTToken("external/gpt-secretkey.txt")
	setupAPI()
	log.Fatal().Err(http.ListenAndServeTLS(
		":8080", "external/server.crt", "external/server.key", nil))
}

/* CodeNames as a Service: If you wanted to make this S C A L E...
   At the point where you start running multiple instances, it is common to include
   Redis or RabbitMQ to allow distributed messages for the websockets.
   You would then listen on the PubSub schema used and push messages on RabbitMQ/Redis,
   then read from those topics and push onto the Websockets. */
func setupAPI() {
	ctx := context.Background()

	readDictionary("./codenames-wordlist.txt")

	manager := NewManager(ctx)

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.serveWS)
	http.HandleFunc("/login", manager.loginHandler)
}

func getGPTToken(path string) string {
	file, err := os.Open(path)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Error opening %v", path))
		return ""
	}
	defer file.Close() // Close the file when we're done

	pword, err := io.ReadAll(file)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Error reading %v", path))
		return ""
	}
	return string(pword)
}
