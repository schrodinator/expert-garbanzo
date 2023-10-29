package main

import (
	"context"
	"log"
	"net/http"
)

func main() {
	setupAPI()

	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
}

// At the point where you start running multiple instances, it is common to include
// Redis or RabbitMQ to allow distributed messages for the websockets.
// You would then listen on the PubSub schema used and push messages on RabbitMQ/Redis,
// then read from those topics and push onto the Websockets.
func setupAPI() {
	ctx := context.Background()

	readDictionary("/usr/share/dict/words")

	manager := NewManager(ctx)

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.serveWS)
	http.HandleFunc("/login", manager.loginHandler)
}
