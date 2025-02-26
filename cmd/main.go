package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"tictacgo/internal/routes"
	"time"
)

func main() {
	// Set up all routes
	routes.SetupRoutes()

	// Handle graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)

	server := &http.Server{Addr: ":8080"}

	go func() {
		slog.Info("Server started at :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("ListenAndServe(): %s", "Error", err)
		}
	}()

	<-interrupt
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server Shutdown Failed:%+v", "Error", err)
	}
	fmt.Println("Server exited properly")
}
