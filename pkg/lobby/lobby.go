package lobby

import (
	"encoding/json" // Used to encode Go data structures into JSON format.
	"fmt"
	"html/template" // Provides functions for parsing and executing HTML templates, allowing the rendering of HTML content with dynamic data.
	"net/http"      // handles http requests
	"tictacgo/pkg/game"
	"tictacgo/pkg/models"

	"github.com/google/uuid" // generate uuids
)

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

	// A slice openLobbies is created
	var openLobbies []*models.Lobby

	// all the lobbies from the models.Lobbies map are added to above slice.
	for _, lobby := range models.Lobbies {
		openLobbies = append(openLobbies, lobby)
	}

	// sets the response's content type to application/json, indicating that the response will contain JSON data.
	w.Header().Set("Content-Type", "application/json")

	// encodes the openLobbies slice into JSON format and sends it as the response to the client.
	json.NewEncoder(w).Encode(openLobbies)
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
