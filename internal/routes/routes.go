package routes

import (
	"net/http"
	"tictacgo/api/handlers"
	"tictacgo/internal/lobby"

	"golang.org/x/net/websocket"
)

// SetupRoutes configures all HTTP routes for the server
func SetupRoutes() {
	// Serve static files from "web/templates"
	http.Handle("/", http.FileServer(http.Dir("./web/templates")))

	// Serve static files from "web/static"
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handle lobby routes
	http.HandleFunc("/create-lobby", lobby.CreateLobby)
	http.HandleFunc("/lobbies", lobby.HandleLobbies)
	http.HandleFunc("/lobby/", lobby.ServeLobby)

	// WebSocket handler
	http.Handle("/ws", websocket.Handler(handlers.HandleWebSocket))
}
