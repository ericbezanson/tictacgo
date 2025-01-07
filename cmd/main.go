package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"tictacgo/pkg/game"
	"tictacgo/pkg/ws"

	"golang.org/x/net/websocket"
)

func main() {
	// Get the current working directory
	basePath, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	// Construct the absolute path to the "web/templates" directory
	templatesPath := filepath.Join(basePath, "web", "templates")

	// Initialize the Tic-Tac-Toe game
	ticTacToe := game.NewGame()

	// Serve static files from the "web/templates" directory
	fs := http.FileServer(http.Dir(templatesPath))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Home page route
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(templatesPath, "index.html"))
	})

	// Lobby route
	http.HandleFunc("/lobby", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(templatesPath, "lobby.html"))
	})

	// WebSocket route
	http.Handle("/ws", websocket.Handler(func(conn *websocket.Conn) {
		ws.HandleConnections(conn, ticTacToe)
	}))

	// Start the server
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
