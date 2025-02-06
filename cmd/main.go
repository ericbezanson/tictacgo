package main

import (
	"context" // Allows you to manage deadlines, cancelation signals, and request-scoped values
	"fmt"
	"log"
	"net/http"
	"os"        // Provides a platform-independent interface to OS functionality.
	"os/signal" //Allows the program to intercept and handle OS signals.
	"tictacgo/api/handlers"
	"tictacgo/pkg/lobby"
	"time" // Provides functionality for measuring and displaying time.

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
	// http.HandleFunc("/lobbies", lobby.GetLobbies)
	http.HandleFunc("/lobbies", lobby.HandleLobbies)

	// WebSocket handler
	http.Handle("/ws", websocket.Handler(handlers.HandleWebSocket))

	// Serve specific lobbies
	http.HandleFunc("/lobby/", lobby.ServeLobby)

	/// ------------------------------------------------------------------------------------------------ Graceful shutdown

	// Creates a channel to receive OS signals, specifically interrupt signals.
	// channels used to set up a communication mechanism between goroutines, which are lightweight threads of execution in Go
	interrupt := make(chan os.Signal, 1) // buffer size = 1 -- can hold one value
	// Subscribes the interrupt channel to receive notifications of OS interrupt signals (AKA killing localhost with Ctrl+C)
	signal.Notify(interrupt, os.Interrupt)

	// 	NOTES
	// 	By calling signal.Notify(interrupt, os.Interrupt), the program tells the Go runtime to send any received os.Interrupt signal to the interrupt channel.
	// When Ctrl+C is pressed, the OS sends an interrupt signal to the program, which the signal.Notify function catches and forwards to the interrupt channel.
	// This setup allows the program to react to the signal asynchronously, meaning it doesn't have to continuously check for the signal in a loop but will be notified as soon as the signal is sent.

	// insures that the program stops listening to interrupt signals when the program terminates.
	defer signal.Stop(interrupt)

	// initializes a new HTTP server that listens on port 8080.
	server := &http.Server{Addr: ":8080"}

	go func() {
		fmt.Println("Server started at :8080")
		// Starts the server and listens for incoming HTTP requests.
		// Checks if the server stops due to an error other than a deliberate shutdown
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err) // Use log.Fatalf here
		}
	}()

	// Blocks the main goroutine until an interrupt signal is received.
	// <- channel recieve opperation
	// prevents the code below from running until OS sends an interupt signal (like CMD+C)
	<-interrupt
	fmt.Println("Shutting down server...")

	// Creates a context with a 5-second timeout, which will be used to shut down the server gracefully.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Ensures the context is canceled when the shutdown is complete, freeing up resources.
	defer cancel()

	// Attempts to gracefully shut down the server within the timeout period specified by the context.
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err) // Use log.Fatalf here
	}
	fmt.Println("Server exited properly")
}
