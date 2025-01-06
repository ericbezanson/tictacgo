package main

import (
	"fmt"
	"net/http"

	"tictacgo/pkg/game"
	"tictacgo/pkg/ws"

	"golang.org/x/net/websocket"
)

func main() {
	ticTacToe := game.NewGame()

	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.Handle("/ws", websocket.Handler(func(conn *websocket.Conn) {
		ws.HandleConnections(conn, ticTacToe)
	}))

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
