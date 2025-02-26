package lobby

import (
	"encoding/json" // Used to encode Go data structures into JSON format.
	"fmt"
	"html/template" // Provides functions for parsing and executing HTML templates, allowing the rendering of HTML content with dynamic data.
	"log"
	"net/http" // handles http requests
	"os"
	"tictacgo/internal/chat"
	"tictacgo/internal/game"
	"tictacgo/models"

	"github.com/go-redis/redis"
	"github.com/google/uuid" // generate uuids
	"golang.org/x/net/websocket"
)

var redisClient = redis.NewClient(&redis.Options{
	Addr: os.Getenv("REDIS_ADDRESS"), // Use environment variable
})

func CreateLobby(w http.ResponseWriter, r *http.Request) {

	username := r.URL.Query().Get("Name")
	println(username)

	if username == "" {
		// Handle the case where no username is passed (optional)
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Read the username from the cookie
	cookie, err := r.Cookie("Name")
	if err == nil { // Only access cookie.Value if there is no error
		username = cookie.Value
	}

	// creates a new instance of a game using a function defined in the game package.
	newGame := game.NewGame()

	// enerates a unique ID for the new lobby using UUID.
	lobbyID := uuid.New().String()

	// A new models.Lobby is created with a unique lobbyID, a name, max of 2 players (MaxPlayers: 2), and the newly created game (Game: newGame).
	// & goes in front of a variable when you want to get that variable's memory address
	newLobby := &models.Lobby{
		ID:           lobbyID,
		Name:         fmt.Sprintf("%s's Lobby", username),
		MaxPlayers:   2,
		Game:         newGame, // Initialize the Game here
		Players:      []*models.Player{},
		ReadyPlayers: make(map[string]bool), // ✅ Initialize the map
		ChatMessages: []models.ChatMessage{},
	}

	// stores the newly created lobby in a global
	models.Lobbies[lobbyID] = newLobby

	// redirects the user to the newly created lobby's page
	http.Redirect(w, r, "/lobby/"+lobbyID, http.StatusSeeOther)
}

func ServeLobby(w http.ResponseWriter, r *http.Request) {
	lobbyID := r.URL.Path[len("/lobby/"):]

	// Check if the lobby exists in memory
	lobby, exists := models.Lobbies[lobbyID]
	if !exists {
		// Fetch from Redis if not in memory
		lobbyData, err := redisClient.Get("lobby:" + lobbyID).Result()
		if err != nil {
			log.Printf("Lobby %s not found in Redis: %v", lobbyID, err)
			http.NotFound(w, r)
			return
		}

		// Create a new pointer for the lobby
		lobby = &models.Lobby{}
		if err := json.Unmarshal([]byte(lobbyData), lobby); err != nil { // <-- FIXED
			log.Printf("Error decoding lobby %s: %v", lobbyID, err)
			http.Error(w, "Failed to load lobby", http.StatusInternalServerError)
			return
		}

		// Store in models.Lobbies so it persists in memory
		models.Lobbies[lobbyID] = lobby
	}

	// Render the lobby page
	tmpl, _ := template.ParseFiles("./web/templates/lobby.html")
	tmpl.Execute(w, lobby)
}

// AssignAndNotifyPlayer assigns a symbol to a player and notifies the lobby
func AssignAndNotifyPlayer(lobby *models.Lobby, ws *websocket.Conn, username string, id string, lobbyConnections map[string][]*websocket.Conn) {

	var player *models.Player

	// Check if an ID is provided
	if id != "" {
		// Search for an existing player with the given ID
		for _, existingPlayer := range lobby.Players {
			if existingPlayer.ID == id {
				// Player found, send assignPlayer message without modifying lobby.Players
				messageToUser := map[string]interface{}{
					"type":     "assignPlayer",
					"username": existingPlayer.Name,
					"symbol":   existingPlayer.Symbol,
					"id":       existingPlayer.ID,
				}
				sendJSON(ws, messageToUser)
				return // Exit function, no further processing needed
			}
		}
		// If ID is provided but no match is found, proceed as new player with given ID
		player = &models.Player{ID: id, Name: username, Ready: false}
	} else {
		// No ID provided, generate a new one
		playerID := uuid.New().String()
		player = &models.Player{ID: playerID, Name: username, Ready: false}
	}

	// Assign symbol based on number of players
	if len(lobby.Players) < 2 {
		symbol := "X"
		if len(lobby.Players) == 1 {
			symbol = "O"
		}
		player.Symbol = symbol
		lobby.Players = append(lobby.Players, player)

		// Notify the new player
		messageToUser := map[string]interface{}{
			"type":     "assignPlayer",
			"username": player.Name,
			"symbol":   player.Symbol,
			"id":       player.ID,
		}
		sendJSON(ws, messageToUser)

		// Notify chat about the new player
		gameMasterMessage := map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   fmt.Sprintf("%v has joined the game!", player.Name),
		}
		chat.HandleChatMessage(lobby.ID, gameMasterMessage, lobbyConnections)

	} else {
		// Assign additional connections as spectators
		player.Symbol = "S"
		lobby.Players = append(lobby.Players, player)

		playerAssign := map[string]interface{}{
			"type":     "assignPlayer",
			"username": player.Name,
			"symbol":   player.Symbol,
			"id":       player.ID,
		}

		sendJSON(ws, playerAssign)
		// Notify spectator
		messageToUser := map[string]interface{}{
			"type":     "lobbyFull",
			"userName": player.Name,
			"text":     "The lobby is full, you are now spectating.",
			"ID":       player.ID,
		}
		sendJSON(ws, messageToUser)

		// Notify chat about the spectator
		gameMasterMessage := map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   fmt.Sprintf("%v is now spectating!", player.Name),
		}
		chat.HandleChatMessage(lobby.ID, gameMasterMessage, lobbyConnections)

	}
}

// Helper function to send a JSON message over the WebSocket
// used for sending to individial client instead of all clients
func sendJSON(ws *websocket.Conn, msg map[string]interface{}) {
	// convert go map data structure into a json-formatted string
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		return
	}
	if _, err := ws.Write(jsonData); err != nil {
		log.Println("Error sending JSON:", err)
	}
}

func fetchLobbiesFromRedis() ([]models.Lobby, error) {
	// Get all keys matching "lobby:*"
	keys, err := redisClient.Keys("lobby:*").Result()
	if err != nil {
		return nil, err
	}

	// create variable to store lobbies in
	var lobbies []models.Lobby
	for _, key := range keys {
		// Fetch data from Redis
		data, err := redisClient.Get(key).Result()
		if err != nil {
			log.Printf("Error fetching %s: %v", key, err)
			continue
		}

		var lobby models.Lobby
		//de-serialize JSON into a models.lobby struct
		if err := json.Unmarshal([]byte(data), &lobby); err != nil {
			log.Printf("Error unmarshalling %s: %v", key, err)
			continue
		}
		// add parsed lobby to list of lobbies
		lobbies = append(lobbies, lobby)
	}

	return lobbies, nil
}

// handler called when /lobbies api is hit
func HandleLobbies(w http.ResponseWriter, r *http.Request) {
	lobbies, err := fetchLobbiesFromRedis()
	if err != nil {
		http.Error(w, "Failed to fetch lobbies", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lobbies)
}
