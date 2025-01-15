package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"tictacgo/api/handlers"
	"tictacgo/pkg/lobby"
	"tictacgo/pkg/models"
	"tictacgo/pkg/redis"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	// Initialize Redis
	redis.Init()

	// Setup HTTP handlers
	http.Handle("/", http.FileServer(http.Dir("./web/templates")))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))
	http.HandleFunc("/create-lobby", lobby.CreateLobby)
	http.HandleFunc("/lobbies", lobby.GetLobbies)
	http.Handle("/ws", websocket.Handler(handlers.HandleWebSocket))
	http.HandleFunc("/lobby/", lobby.ServeLobby)

	// Graceful shutdown setup
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)

	server := &http.Server{Addr: ":8080"}
	go func() {
		fmt.Println("Server started at :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()

	<-interrupt
	fmt.Println("Shutting down server...")

	// Clear connections before shutting down
	clearLobbyConnections()

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	fmt.Println("Server exited properly")
}

func clearLobbyConnections() {
	fmt.Println("Clearing Lobby Connections...", len(models.Lobbies))

	err := redis.UpdateLobby()
	if err != nil {
		log.Printf("Failed to update lobby %s in Redis: ", err)
	}

}
