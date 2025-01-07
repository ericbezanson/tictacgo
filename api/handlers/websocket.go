package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"tictacgo/pkg/models"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

// globals
// store active game lobbies in a map, a map in go is used to store key value pairs
var Lobbies = make(map[string]*models.Lobby)

// A mutex (short for mutual exclusion lock), used to synchronize access to shared resources. This prevents concurrent access to shared data, ensuring that only one thread can modify or read from Lobbies at a time, avoiding race conditions.
var Mu sync.Mutex

// HandleWebSocket - Handle WebSocket connection
func HandleWebSocket(ws *websocket.Conn) {
	// Extract lobby ID from the query string
	query := ws.Request().URL.Query()
	lobbyID := query.Get("lobby")

	// Check if lobbyID is empty
	if lobbyID == "" {
		fmt.Println("Lobby ID is required")
		ws.Close()
		return
	}

	// Lock and check if the lobby exists
	Mu.Lock()
	lobby, exists := models.Lobbies[lobbyID]
	Mu.Unlock()

	// Log the check
	fmt.Printf("Lobby exists: %v Lobby data: %+v\n", exists, lobby)

	// If the lobby doesn't exist, close the WebSocket connection
	if !exists {
		fmt.Println("Lobby not found, closing connection.")
		ws.Close()
		return
	}

	// Proceed with assigning players to the lobby
	playerID := uuid.New().String()
	player := &models.Player{ID: playerID}

	// Assign players based on the current player count
	if len(lobby.Players) < 2 {
		// Assign the first two players as "X" and "O"
		symbol := "X"
		if len(lobby.Players) == 1 {
			symbol = "O"
		}
		player.Symbol = symbol // Assign the correct symbol
		player.Name = fmt.Sprintf("Player %s", symbol)
		lobby.Players = append(lobby.Players, player)

		// Prepare the message to send to the client
		msg := map[string]interface{}{
			"type":     "assignPlayer",
			"userName": player.Name,
			"symbol":   player.Symbol,
		}

		// Marshal the message to JSON
		jsonData, err := json.Marshal(msg)
		if err != nil {
			log.Println("Error marshalling JSON:", err)
			return
		}

		// Send the JSON message as a raw WebSocket message (using Write)
		if _, err := ws.Write(jsonData); err != nil {
			log.Println("Error sending JSON:", err)
		}
	} else {
		// Assign all additional connections as spectators
		player.Symbol = "Spectator" // Spectator symbol
		player.Name = fmt.Sprintf("Spectator %d", len(lobby.Players)-1)
		lobby.Players = append(lobby.Players, player)

		// Prepare the message for spectators
		msg := map[string]interface{}{
			"type":     "lobbyFull",
			"userName": player.Name,
			"text":     "The lobby is full, you are now spectating.",
		}

		// Marshal the message to JSON
		// function from the encoding/json package that is used to convert a Go object (e.g., a struct, map, or slice) into a JSON-encoded byte slice
		jsonData, err := json.Marshal(msg)
		if err != nil {
			log.Println("Error marshalling JSON:", err)
			return
		}

		// Send the JSON message as a raw WebSocket message (using Write)
		if _, err := ws.Write(jsonData); err != nil {
			log.Println("Error sending JSON:", err)
		}
	}

	// Add the WebSocket connection to the lobby
	lobby.Conns = append(lobby.Conns, ws)

	// Broadcast the updated lobby state
	BroadcastLobbyState(lobby)

	// Handle incoming messages
	for {
		var msg map[string]interface{}
		err := websocket.JSON.Receive(ws, &msg) // or you can use ReadMessage directly if needed
		if err != nil {
			break
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "chat":
			BroadcastChatMessage(lobby, msg)
		case "move":
			BroadcastGameMove(lobby, msg)
		}
	}

	// Clean up connection on disconnect
	RemoveConnection(lobby, ws)
	BroadcastLobbyState(lobby)
}

func BroadcastLobbyState(lobby *models.Lobby) {
	state := struct {
		Type    string           `json:"type"`
		Players []*models.Player `json:"players"`
	}{
		Type:    "updatePlayers",
		Players: lobby.Players,
	}

	Mu.Lock()
	for _, conn := range lobby.Conns {
		websocket.JSON.Send(conn, state)
	}
	Mu.Unlock()
}

func BroadcastChatMessage(lobby *models.Lobby, msg map[string]interface{}) {
	// Log the incoming chat message
	fmt.Printf("Broadcasting chat message: %+v\n", msg)

	Mu.Lock()
	for _, conn := range lobby.Conns {
		if err := websocket.JSON.Send(conn, msg); err != nil {
			fmt.Printf("Error broadcasting message: %v\n", err)
		}
	}
	Mu.Unlock()
}

func BroadcastGameMove(lobby *models.Lobby, msg map[string]interface{}) {
	// Log the incoming move message
	log.Printf("Received move: %+v", msg)

	// Get the game associated with the lobby
	game := lobby.Game

	// Extract the move details
	position := msg["position"].(int)
	symbol := msg["symbol"].(string)

	// Make the move
	if game.MakeMove(position, symbol) {
		// Log successful move
		log.Printf("Move made at position %d by symbol %s", position, symbol)

		// Switch the turn
		game.SwitchTurn()

		// Broadcast the move to all players
		Mu.Lock()
		for _, conn := range lobby.Conns {
			if err := websocket.JSON.Send(conn, msg); err != nil {
				log.Printf("Error sending move message: %v", err)
				continue
			}
		}
		Mu.Unlock()

		// Broadcast the updated turn info
		updateTurnMsg := map[string]interface{}{
			"type": "updateTurn",
			"text": game.CurrentTurn,
		}
		BroadcastChatMessage(lobby, updateTurnMsg)
	} else {
		// Log failure to make a valid move
		log.Printf("Invalid move: Position %d already filled or out of bounds", position)
	}
}

func RemoveConnection(lobby *models.Lobby, conn *websocket.Conn) {
	for i, c := range lobby.Conns {
		if c == conn {
			lobby.Conns = append(lobby.Conns[:i], lobby.Conns[i+1:]...)
			break
		}
	}
}
