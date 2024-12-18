package main

import (
	"fmt"
	"log"
	"net/http"

	"tictacgo/game"
	"tictacgo/ws"

	"golang.org/x/net/websocket"
)

func main() {
	// Initialize the Tic-Tac-Toe game
	ticTacToe := game.NewGame()

	// Serve static files from the "static" directory
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Home page route
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	// Lobby route
	http.HandleFunc("/lobby", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/lobby.html")
	})

	// WebSocket route
	http.Handle("/ws", websocket.Handler(func(conn *websocket.Conn) {
		ws.HandleConnections(conn, ticTacToe)
	}))

	// Start the server
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
