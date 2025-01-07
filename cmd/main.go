package main

import (
	"fmt"
	"net/http"
	"tictacgo/api/handlers"
	"tictacgo/pkg/lobby"

	"golang.org/x/net/websocket"
)

func main() {
	// Serve static files from "web/templates"
	http.Handle("/", http.FileServer(http.Dir("./web/templates")))

	// Serve static files from "web/static"
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handle requests to /create-lobby and /lobbies
	http.HandleFunc("/create-lobby", lobby.CreateLobby)
	http.HandleFunc("/lobbies", lobby.GetLobbies)

	// WebSocket handler
	http.Handle("/ws", websocket.Handler(handlers.HandleWebSocket))

	// Serve specific lobbies
	http.HandleFunc("/lobby/", lobby.ServeLobby)

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
