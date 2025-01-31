package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"tictacgo/models"
	"tictacgo/pkg/chat"
	"time"

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

	// if lobby exists, init a new state with an empty gameboard
	if exists {
		lobby.Game.Board = [9]string{}
		// Check if ChatMessages is already initialized, otherwise initialize as empty array
		if lobby.ChatMessages == nil {
			lobby.ChatMessages = []models.ChatMessage{}
		}
		// Check if ReadyPlayers is already initialized, otherwise initialize as empty map
		if lobby.ReadyPlayers == nil {
			lobby.ReadyPlayers = make(map[string]bool)
		}
	}

	// Log the check result for debugging
	fmt.Printf("Lobby exists: %v Lobby data: %+v\n", exists, lobby)

	////// ------------------------------------------------------------------------------------------------------------- Player Assign

	// Proceed with assigning players to the lobby
	playerID := uuid.New().String()
	// A new player object is created with this unique ID, using Player struct in models.go
	player := &models.Player{ID: playerID}

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
		case "setUsername":
			if username, ok := msg["userName"].(string); ok {
				// Now you can use the username as needed
				player.Name = username
				assignAndNotifyPlayer(lobby, ws, player)
			} else {
				fmt.Println("Username is missing or not a string in the received message.")
			}
		case "chat":
			chat.HandleChatMessage(ws, lobby, player, msg)
		case "move":
			// Extract the move details
			rawPosition := msg["position"].(float64)
			username := msg["userName"].(string)
			// convert based on format game needs
			// TODO! - update so that game and data coming from server have matching types
			position := int(rawPosition)
			symbol := msg["symbol"].(string)

			// BroadcastGameMove(lobby, ws, msg)
			response := lobby.Game.HandleGameMove(position, symbol, username)
			fmt.Println("resp", response)
			// sendJSON(ws, response) // Send result back to client
			for _, conn := range lobby.Conns {
				if err := websocket.JSON.Send(conn, response); err != nil {
					log.Printf("Error sending move message: %v", err)
					continue
				}
			}
		case "ready":
			// Mark the player as ready
			// add player to ReadyPlayers map
			// TODO!- add support for unready and check bool values
			fmt.Println("ready", player)
			lobby.ReadyPlayers[player.ID] = true

			// Check if both players are ready to start the game
			if len(lobby.ReadyPlayers) == 2 {

				// Broadcast startGame message to all connected clients
				start := map[string]interface{}{
					"type": "startGame",
				}

				// send statGame message to all Conns in the lobby
				for _, conn := range lobby.Conns {
					sendJSON(conn, start)
				}

				// Notify both players that the game is ready to start
				BroadcastChatMessage(lobby, map[string]interface{}{
					"type":   "chat",
					"sender": "GAMEMASTER",
					"text":   "Both players are ready. The game will start now!",
				})

				// Broadcast lobby state and notify other players
				// BroadcastLobbyState(lobby)
			}
		case "unready":
			fmt.Println("player unready: ", player)

			fmt.Println("player unready: ", player)
			delete(lobby.ReadyPlayers, player.ID)
			// Remove the player from the ReadyPlayers map
			BroadcastLobbyState(lobby) // Broadcast the updated state after removing the player
		}

	}
}

////// ------------------------------------------------------------------------------------------------------------- Helpers

// broadcast state to a newly connected user when they first connect to the lobby
func HandleInitialConnection(ws *websocket.Conn, lobby *models.Lobby) {
	initialState := struct {
		Type         string               `json:"type"`
		GameBoard    [9]string            `json:"gameBoard"`
		CurrentTurn  string               `json:"currentTurn"`
		GameStarted  bool                 `json:"gameStarted"`
		ChatMessages []models.ChatMessage `json:"chatMessages"`
		ReadyPlayers map[string]bool      `json:"readyPlayers"`
	}{
		Type:         "initialState",
		GameBoard:    lobby.Game.Board,
		CurrentTurn:  lobby.CurrentTurn,
		GameStarted:  lobby.GameStarted,
		ChatMessages: lobby.ChatMessages,
		ReadyPlayers: lobby.ReadyPlayers,
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
		Type         string               `json:"type"`
		GameBoard    [9]string            `json:"gameBoard"`
		CurrentTurn  string               `json:"currentTurn"`
		GameStarted  bool                 `json:"gameStarted"`
		ChatMessages []models.ChatMessage `json:"chatMessages"`
		ReadyPlayers map[string]bool      `json:"readyPlayers"`
	}{
		Type:         "updatePlayers",
		GameBoard:    lobby.Game.Board,
		CurrentTurn:  lobby.CurrentTurn,
		GameStarted:  lobby.GameStarted,
		ChatMessages: lobby.ChatMessages,
		ReadyPlayers: lobby.ReadyPlayers,
	}

	// Send state to all connected clients
	for _, conn := range lobby.Conns {
		fmt.Println("conn mesg sent", conn)
		websocket.JSON.Send(conn, state)
	}
}

////// ------------------------------------------------------------------------------------------------------------- Chat Helpers

func BroadcastChatMessage(lobby *models.Lobby, msg map[string]interface{}) {
	fmt.Printf("Broadcasting chat message: %+v\n", msg)

	// Create a chat message
	chatMessage := models.ChatMessage{
		Text:      msg["text"].(string),
		Sender:    msg["sender"].(string),
		Timestamp: time.Now(),
	}

	// Add the message to the lobby's chat messages array
	if lobby != nil {
		lobby.ChatMessages = append(lobby.ChatMessages, chatMessage)
	} else {
		fmt.Println("Lobby is nil. Cannot append chat message.")
		return
	}

	// Broadcast the updated lobby state (including the updated chat messages)
	BroadcastLobbyState(lobby)
}

////// ------------------------------------------------------------------------------------------------------------- Game Helpers

func assignAndNotifyPlayer(lobby *models.Lobby, ws *websocket.Conn, player *models.Player) {

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
			"text":   fmt.Sprintf("%v has joined the game!", player.Name),
		})

	} else {
		// Assign additional connections as spectators
		player.Symbol = "S" // Spectator symbol
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
			"text":   fmt.Sprintf("Spectator %v is now spectating!", player.Name),
		})
	}
}
