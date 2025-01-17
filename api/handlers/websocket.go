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
// store active game lobbies in a map, a map in go is used to store key value pairs, dictionary-like
var Lobbies = make(map[string]*models.Lobby)

// HandleWebSocket - Handle WebSocket connection
// manages Chat, Moves, Ready Messages, and Game State
func HandleWebSocket(ws *websocket.Conn) {

	////// ------------------------------------------------------------------------------------------------------------- SETUP

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
	// looked up by using the lobby ID as the map key
	lobby, exists := models.Lobbies[lobbyID]

	// if lobby exists but state is nil, init a new state with an empty gameboard
	if exists && lobby.State == nil {
		lobby.State = &models.LobbyState{
			GameBoard:    [9]string{},
			ChatMessages: []models.Message{},
			Players:      []string{},
			ReadyPlayers: map[string]bool{}, // Track readiness of players
		}
	}

	// Log the check result for debugging
	fmt.Printf("Lobby exists: %v Lobby data: %+v\n", exists, lobby)

	////// ------------------------------------------------------------------------------------------------------------- Player Assign

	// Proceed with assigning players to the lobby
	playerID := uuid.New().String()
	// A new player object is created with this unique ID, using Player struct in models.go
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

		// Prepare and send the message to individual player
		// used for displaying a players username only in their client
		gameMasterMsg := map[string]interface{}{
			"type":     "assignPlayer",
			"userName": player.Name,
			"symbol":   player.Symbol,
		}

		// send to individual users client
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
		// used to populate text in lobby full alert message
		msg := map[string]interface{}{
			"type":     "lobbyFull",
			"userName": player.Name,
			"text":     "The lobby is full, you are now spectating.",
		}
		// send to specific ws Conn
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

	//!TODO - investigate ways to simplify this into one function, avoid race
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

		////// ------------------------------------------------------------------------------------------------------------- Player Actions
		// Handle message based on its type
		switch msgType {
		case "chat":
			BroadcastChatMessage(lobby, msg)
		case "move":
			BroadcastGameMove(lobby, ws, msg)
		case "ready":
			// Mark the player as ready
			// add player to ReadyPlayers map
			// TODO!- add support for unready and check bool values
			lobby.State.ReadyPlayers[player.ID] = true

			// Check if both players are ready to start the game
			if len(lobby.State.ReadyPlayers) == 2 {

				// Broadcast startGame message to all connected clients
				startGameMsg := map[string]interface{}{
					"type": "startGame",
				}

				// send statGame message to all Conns in the lobby
				for _, conn := range lobby.Conns {
					sendJSON(conn, startGameMsg)
				}

				// Notify both players that the game is ready to start
				BroadcastChatMessage(lobby, map[string]interface{}{
					"type":   "startGame",
					"sender": "GAMEMASTER",
					"text":   "Both players are ready. The game will start now!",
				})
			}
		}
	}
}

////// ------------------------------------------------------------------------------------------------------------- Helpers

// broadcast state to a newly connected user when they first connect to the lobby
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

// updates all connected clients with the current lobby state
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

////// ------------------------------------------------------------------------------------------------------------- Chat Helpers

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

////// ------------------------------------------------------------------------------------------------------------- Game Helpers

func BroadcastGameMove(lobby *models.Lobby, ws *websocket.Conn, msg map[string]interface{}) {
	// Log the incoming move message
	log.Printf("Received move: %+v", msg)

	// Get the game associated with the lobby
	game := lobby.Game

	// Extract the move details
	rawPosition := msg["position"].(float64)
	// convert based on format game needs
	// TODO! - update so that game and data coming from server have matching types
	positionAsInt := int(rawPosition)
	symbol := msg["symbol"].(string)

	// check if game accepts incoming move as a valid one, then execute logic
	if game.MakeMove(positionAsInt, symbol) {

		// !NOTE - proper sequence of events:
		// Player makes a move (server processes the move).
		// UI is updated (via WebSocket message).
		// Win or draw condition is checked (server sends an alert and chat message).
		// If no win or draw, call SwitchTurn() (server updates the turn and notifies).

		// Log successful move
		log.Printf("Move made at position %d by symbol %s", positionAsInt, symbol)

		// update gameboard in LobbyState
		lobby.State.GameBoard[positionAsInt] = symbol

		// Broadcast the move to all players (UI update)
		for _, conn := range lobby.Conns {
			if err := websocket.JSON.Send(conn, msg); err != nil {
				log.Printf("Error sending move message: %v", err)
				continue
			}
		}

		// handle outcome of move (win, stalemate or standard move)
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
