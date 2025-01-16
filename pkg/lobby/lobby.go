package lobby

import (
	"context"
	"encoding/json" // Used to encode Go data structures into JSON format.
	"fmt"
	"html/template" // Provides functions for parsing and executing HTML templates, allowing the rendering of HTML content with dynamic data.
	"net/http"      // handles http requests
	"tictacgo/pkg/game"
	"tictacgo/pkg/models"
	"tictacgo/pkg/redis"

	redisDirect "github.com/go-redis/redis/v8"
	"github.com/google/uuid" // generate uuids
)

var ctx = context.Background()

func CreateLobby(w http.ResponseWriter, r *http.Request) {
	// creates a new instance of a game using a function defined in the game package.
	newGame := game.NewGame()

	// enerates a unique ID for the new lobby using UUID.
	lobbyID := uuid.New().String()

	// A new models.Lobby is created with a unique lobbyID, a name, max of 2 players (MaxPlayers: 2), and the newly created game (Game: newGame).
	// & goes in front of a variable when you want to get that variable's memory address
	newLobby := &models.Lobby{
		ID:         lobbyID,
		Name:       fmt.Sprintf("Lobby %s", lobbyID),
		MaxPlayers: 2,
		Game:       newGame, // Initialize the Game here
	}

	// stores the newly created lobby in a global
	models.Lobbies[lobbyID] = newLobby

	// redirects the user to the newly created lobby's page
	http.Redirect(w, r, "/lobby/"+lobbyID, http.StatusSeeOther)
}

func GetLobbies(w http.ResponseWriter, r *http.Request) {
	var openLobbies []*models.Lobby

	// Check Redis for stored lobbies
	iter := redis.Client.Scan(ctx, 0, "lobby:*", 0).Iterator()
	for iter.Next(ctx) {
		lobbyID := iter.Val()
		lobbyData, err := redis.Client.Get(ctx, lobbyID).Result()
		if err == redisDirect.Nil {
			continue // Key does not exist, continue to the next
		} else if err != nil {
			http.Error(w, "Failed to fetch lobbies from Redis", http.StatusInternalServerError)
			return // Handle other errors here
		}

		var lobby models.Lobby
		err = json.Unmarshal([]byte(lobbyData), &lobby)
		if err != nil {
			http.Error(w, "Failed to parse lobby data", http.StatusInternalServerError)
			return // Handle parsing errors here
		}
		openLobbies = append(openLobbies, &lobby)
		models.Lobbies[lobby.ID] = &lobby
	}

	// Include existing in-memory lobbies
	for _, lobby := range models.Lobbies {
		openLobbies = append(openLobbies, lobby)
	}

	// Check for errors from the iterator
	if err := iter.Err(); err != nil {
		http.Error(w, "Failed to iterate Redis keys", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(openLobbies) == 0 {
		// Return an empty array if no lobbies exist
		w.Write([]byte("[]"))
		return
	}

	err := json.NewEncoder(w).Encode(openLobbies)
	if err != nil {
		http.Error(w, "Failed to fetch lobbies", http.StatusInternalServerError)
		return
	}
}

func ServeLobby(w http.ResponseWriter, r *http.Request) {
	// Extract the lobby ID from the URL path.
	lobbyID := r.URL.Path[len("/lobby/"):]

	// First, look for the lobby in models.Lobbies.
	lobby, exists := models.Lobbies[lobbyID]

	fmt.Println("LobbyID", lobbyID)
	fmt.Println("Lobby", lobby)

	if !exists {
		// If the lobby is not in the in-memory map, check Redis.
		lobbyData, err := redis.Client.Get(ctx, "lobby:"+lobbyID).Result()
		if err == redisDirect.Nil {
			// If the key does not exist in Redis, return a 404 error.
			http.NotFound(w, r)
			return
		} else if err != nil {
			// Handle any other errors when fetching from Redis.
			http.Error(w, "Failed to fetch lobby from Redis", http.StatusInternalServerError)
			return
		}

		// Deserialize the lobby data.
		err = json.Unmarshal([]byte(lobbyData), &lobby)
		if err != nil {
			http.Error(w, "Failed to parse lobby data", http.StatusInternalServerError)
			return
		}
	}

	// Ensure lobby is not nil before proceeding
	if lobby == nil {
		http.NotFound(w, r)
		return
	}

	// If the lobby exists, load the HTML template for the lobby page.
	tmpl, err := template.ParseFiles("./web/templates/lobby.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	// Render the lobby.html template and pass the lobby object to it.
	tmpl.Execute(w, lobby)
}
func clearLobbyConnections() {
	fmt.Println("Clearing Lobby Connections...", len(models.Lobbies))
	err := redis.UpdateLobby()
	if err != nil {
		fmt.Printf("Failed to update lobby %s in Redis: %v", err)
	}
}
