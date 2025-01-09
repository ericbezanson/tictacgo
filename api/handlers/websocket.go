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

	// Retrieve the lobby from the Lobbies map and check if it exists
	lobby, exists := models.Lobbies[lobbyID]

	if exists && lobby.State == nil {
		lobby.State = &models.LobbyState{
			GameBoard:    [9]string{},
			ChatMessages: []models.Message{},
			Players:      []string{},
		}
	}

	// Log the check result for debugging
	fmt.Printf("Lobby exists: %v Lobby data: %+v\n", exists, lobby)

	// If the lobby doesn't exist, create a new lobby state
	if !exists {
		fmt.Println("Lobby not found, initializing a new lobby.")
		lobby = &models.Lobby{
			ID: lobbyID,
			State: &models.LobbyState{
				GameBoard:    [9]string{},        // Initialize an empty game board
				ChatMessages: []models.Message{}, // Initialize an empty chat history
				Players:      []string{},         // Initialize an empty player list
			},
		}
		models.Lobbies[lobbyID] = lobby
	}

	// Proceed with assigning players to the lobby
	playerID := uuid.New().String()
	player := &models.Player{ID: playerID}

	// Check the number of players in the lobby to assign roles
	if len(lobby.Players) < 2 {
		// Assign the first two players as "X" and "O"
		symbol := "X"
		if len(lobby.Players) == 1 {
			symbol = "O"
		}
		player.Symbol = symbol
		player.Name = fmt.Sprintf("Player %s", symbol)
		lobby.Players = append(lobby.Players, player)

		fmt.Println("player", player)

		// Prepare and send the message to the player
		gameMasterMsg := map[string]interface{}{
			"type":     "assignPlayer",
			"userName": player.Name,
			"symbol":   player.Symbol,
			"sender":   "GAMEMASTER",
			"text":     fmt.Sprintf("Player %s has joined the game!", symbol),
		}
		sendJSON(ws, gameMasterMsg)

		// Notify all about the new player joining
		BroadcastChatMessage(lobby, map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   fmt.Sprintf("Player %s has joined the game!", symbol),
		})

		// Broadcast lobby state and notify other players
		BroadcastLobbyState(lobby)
	} else {
		// Assign additional connections as spectators
		player.Symbol = "S" // Spectator symbol
		player.Name = fmt.Sprintf("Spectator %d", len(lobby.Players)-1)
		lobby.Players = append(lobby.Players, player)

		// Prepare and send the message to the spectator
		msg := map[string]interface{}{
			"type":     "lobbyFull",
			"userName": player.Name,
			"text":     "The lobby is full, you are now spectating.",
		}
		sendJSON(ws, msg)

		// Notify all about the new spectator joining
		BroadcastChatMessage(lobby, map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   fmt.Sprintf("Spectator %d is now spectating!", len(lobby.Players)-1),
		})

		// Broadcast lobby state and notify other players
		BroadcastLobbyState(lobby)
	}

	// Add the WebSocket connection to the lobby
	lobby.Conns = append(lobby.Conns, ws)

	// Send the current state to the newly connected client
	HandleInitialConnection(ws, lobby)

	// Broadcast the updated lobby state
	BroadcastLobbyState(lobby)

	// Continuously handle incoming messages
	for {
		var msg map[string]interface{}
		err := websocket.JSON.Receive(ws, &msg)
		if err != nil {
			fmt.Printf("Error receiving message: %v\n", err)
			break
		}
		fmt.Printf("Received message: %+v\n", msg)

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		// Handle message based on its type
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

// Helper function to send a JSON message over the WebSocket
func sendJSON(ws *websocket.Conn, msg map[string]interface{}) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		return
	}
	if _, err := ws.Write(jsonData); err != nil {
		log.Println("Error sending JSON:", err)
	}
}

func BroadcastLobbyState(lobby *models.Lobby) {
	state := struct {
		Type  string             `json:"type"`
		State *models.LobbyState `json:"state"`
	}{
		Type:  "updatePlayers",
		State: lobby.State,
	}

	// Send state to all connected clients
	for _, conn := range lobby.Conns {
		websocket.JSON.Send(conn, state)
	}
}

///----------------------- CHAT

func BroadcastChatMessage(lobby *models.Lobby, msg map[string]interface{}) {
	fmt.Printf("Broadcasting chat message: %+v\n", msg)
	chatMessage := models.Message{
		Type:   msg["type"].(string),
		Text:   msg["text"].(string),
		Sender: msg["sender"].(string),
	}

	if lobby.State != nil {
		lobby.State.ChatMessages = append(lobby.State.ChatMessages, chatMessage)
	} else {
		fmt.Println("Lobby state is nil. Cannot append chat message.")
	}

	for _, conn := range lobby.Conns {
		if err := websocket.JSON.Send(conn, msg); err != nil {
			fmt.Printf("Error broadcasting message to %v: %v\n", conn.Request().RemoteAddr, err)
		} else {
			fmt.Printf("Successfully sent message to %v: %+v\n", conn.Request().RemoteAddr, msg)
		}
	}
}

// --------------------------- GAME

func BroadcastGameMove(lobby *models.Lobby, ws *websocket.Conn, msg map[string]interface{}) {
	// Log the incoming move message
	log.Printf("Received move: %+v", msg)

	// Get the game associated with the lobby
	game := lobby.Game

	// Extract the move details
	rawPosition := msg["position"].(float64)
	positionAsInt := int(rawPosition)
	symbol := msg["symbol"].(string)
	// position := int(msg["position"].(float64))

	// Make the move
	if game.MakeMove(positionAsInt, symbol) {

		// !NOTE - proper sequence of events:
		// Player makes a move (server processes the move).
		// UI is updated (via WebSocket message).
		// Win or draw condition is checked (server sends an alert and chat message).
		// If no win or draw, call SwitchTurn() (server updates the turn and notifies).

		// Log successful move
		log.Printf("Move made at position %d by symbol %s", positionAsInt, symbol)

		lobby.State.GameBoard[positionAsInt] = symbol

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
				"sender": "GAMEMASTER",
				"winner": symbol,
			}
			BroadcastChatMessage(lobby, winMsg)
			game.Reset()
		} else if len(winPatterns) == 0 && game.CheckStalemate() { // Check for a Stalemate
			drawMsg := map[string]interface{}{
				"type":   "draw",
				"text":   "It's a draw!",
				"sender": "GAMEMASTER",
			}
			BroadcastChatMessage(lobby, drawMsg)
			game.Reset()
		} else if len(winPatterns) == 0 && !game.CheckStalemate() { // Standard move
			game.SwitchTurn()
			updateTurnMsg := map[string]interface{}{
				"type":   "updateTurn",
				"text":   game.CurrentTurn,
				"sender": "GAMEMASTER",
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

func HandleInitialConnection(ws *websocket.Conn, lobby *models.Lobby) {
	initialState := struct {
		Type  string             `json:"type"`
		State *models.LobbyState `json:"state"`
	}{
		Type:  "initialState",
		State: lobby.State,
	}

	websocket.JSON.Send(ws, initialState)
}

func RemoveConnection(lobby *models.Lobby, conn *websocket.Conn) {
	for i, c := range lobby.Conns {
		if c == conn {
			lobby.Conns = append(lobby.Conns[:i], lobby.Conns[i+1:]...)
			break
		}
	}
}
