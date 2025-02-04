package lobby

import (
	"encoding/json" // Used to encode Go data structures into JSON format.
	"fmt"
	"html/template" // Provides functions for parsing and executing HTML templates, allowing the rendering of HTML content with dynamic data.
	"log"
	"net/http" // handles http requests
	"tictacgo/models"
	"tictacgo/pkg/chat"
	"tictacgo/pkg/game"

	"github.com/google/uuid" // generate uuids
	"golang.org/x/net/websocket"
)

func CreateLobby(w http.ResponseWriter, r *http.Request) {

	username := r.URL.Query().Get("username")
	if username == "" {
		// Handle the case where no username is passed (optional)
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Now you can use the username in your logic, for example:
	fmt.Println("Username from URL:", username)

	// Read the username from the cookie
	cookie, err := r.Cookie("username")
	if err != nil {
		http.Error(w, "Username cookie not found", http.StatusBadRequest)
		return
	}
	username = cookie.Value

	// Now you can use the username in your logic
	fmt.Println("Username from cookie:", username)

	// creates a new instance of a game using a function defined in the game package.
	newGame := game.NewGame()

	// enerates a unique ID for the new lobby using UUID.
	lobbyID := uuid.New().String()

	// A new models.Lobby is created with a unique lobbyID, a name, max of 2 players (MaxPlayers: 2), and the newly created game (Game: newGame).
	// & goes in front of a variable when you want to get that variable's memory address
	newLobby := &models.Lobby{
		ID:         lobbyID,
		Name:       fmt.Sprintf("%s's Lobby", username),
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

	for _, lobby := range models.Lobbies {
		openLobbies = append(openLobbies, lobby)
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

	// extracts the lobby ID from the URL path.
	lobbyID := r.URL.Path[len("/lobby/"):]

	// The code looks for the lobby in models.Lobbies using lobbyID.
	lobby, exists := models.Lobbies[lobbyID]

	if !exists {
		http.NotFound(w, r)
		return
	}

	// If the lobby exists, the code loads the HTML template for the lobby page
	tmpl, _ := template.ParseFiles("./web/templates/lobby.html")
	// renders the lobby.html template and passes the lobby object to it
	tmpl.Execute(w, lobby)
}

// AssignAndNotifyPlayer assigns a symbol to a player and notifies the lobby
func AssignAndNotifyPlayer(lobby *models.Lobby, ws *websocket.Conn, player *models.Player) {

	// Check the number of players in the lobby to assign roles
	if len(lobby.Players) < 2 {
		// Assign the first two players as "X" and "O"
		symbol := "X"
		if len(lobby.Players) == 1 {
			symbol = "O"
		}
		player.Symbol = symbol
		lobby.Players = append(lobby.Players, player)

		// Prepare and send the message to individual player
		messageToUser := map[string]interface{}{
			"type":     "assignPlayer",
			"userName": player.Name,
			"symbol":   player.Symbol,
		}
		sendJSON(ws, messageToUser)

		// Notify both players
		gameMasterMessage := map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   fmt.Sprintf("%v has joined the game!", player.Name),
		}
		chat.HandleChatMessage(lobby, gameMasterMessage)

	} else {
		// Assign additional connections as spectators
		player.Symbol = "S"
		lobby.Players = append(lobby.Players, player)

		// Send spectator notification
		messageToUser := map[string]interface{}{
			"type":     "lobbyFull",
			"userName": player.Name,
			"text":     "The lobby is full, you are now spectating.",
		}
		sendJSON(ws, messageToUser)

		gameMasterMessage := map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   fmt.Sprintf("%v is now spectating!", player.Name),
		}
		chat.HandleChatMessage(lobby, gameMasterMessage)

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
