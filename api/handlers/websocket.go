package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"tictacgo/pkg/models"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

// globals
// store active game lobbies in a map, a map in go is used to store key value pairs
var Lobbies = make(map[string]*models.Lobby)

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

	// check if the lobby exists

	lobby, exists := models.Lobbies[lobbyID]

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
			BroadcastGameMove(lobby, ws, msg)
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

	for _, conn := range lobby.Conns {
		websocket.JSON.Send(conn, state)
	}

}

func BroadcastChatMessage(lobby *models.Lobby, msg map[string]interface{}) {
	// Log the incoming chat message
	fmt.Printf("Broadcasting chat message: %+v\n", msg)

	for _, conn := range lobby.Conns {
		if err := websocket.JSON.Send(conn, msg); err != nil {
			fmt.Printf("Error broadcasting message: %v\n", err)
		}
	}

}

func BroadcastGameMove(lobby *models.Lobby, ws *websocket.Conn, msg map[string]interface{}) {
	// Log the incoming move message
	log.Printf("Received move: %+v", msg)

	// Get the game associated with the lobby
	game := lobby.Game

	// Extract the move details
	rawPosition := msg["position"].(float64)
	positionAsInt := int(rawPosition)
	symbol := msg["symbol"].(string)

	// Make the move
	if game.MakeMove(positionAsInt, symbol) {

		// !NOTE - proper sequence of events:
		// Player makes a move (server processes the move).
		// UI is updated (via WebSocket message).
		// Win or draw condition is checked (server sends an alert and chat message).
		// If no win or draw, call SwitchTurn() (server updates the turn and notifies).

		// Log successful move
		log.Printf("Move made at position %d by symbol %s", positionAsInt, symbol)

		// Broadcast the move to all players (UI update)
		for _, conn := range lobby.Conns {
			if err := websocket.JSON.Send(conn, msg); err != nil {
				log.Printf("Error sending move message: %v", err)
				continue
			}
		}

		winPatterns := game.CheckWin(symbol)
		if len(winPatterns) > 0 { // Check for a win
			winMsg := map[string]interface{}{
				"type":   "win",
				"text":   fmt.Sprintf("Player %s wins!", symbol),
				"winner": symbol,
			}
			BroadcastChatMessage(lobby, winMsg)
			game.Reset()
		} else if len(winPatterns) == 0 && game.CheckStalemate() { // Check for a Stalemate
			drawMsg := map[string]interface{}{
				"type": "draw",
				"text": "It's a draw!",
			}
			BroadcastChatMessage(lobby, drawMsg)
			game.Reset()
		} else if len(winPatterns) == 0 && !game.CheckStalemate() { // Standard move
			game.SwitchTurn()
			updateTurnMsg := map[string]interface{}{
				"type": "updateTurn",
				"text": game.CurrentTurn,
			}
			BroadcastChatMessage(lobby, updateTurnMsg)
		}

	} else {
		// Send an error message back to the player
		errorMsg := map[string]interface{}{
			"type":     "invalidMove",
			"text":     "Invalid move: Position already filled or out of bounds",
			"position": positionAsInt,
		}
		if err := websocket.JSON.Send(ws, errorMsg); err != nil {
			log.Printf("Error sending invalid move message: %v", err)
		}
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
