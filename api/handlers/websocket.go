package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"tictacgo/models"
	"tictacgo/pkg/chat"
	"tictacgo/pkg/game"
	"tictacgo/pkg/lobby"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

// globals
// store active game lobbies in a map, a map in go is used to store key value pairs, dictionary-like
var Lobbies = make(map[string]*models.Lobby)

// HandleWebSocket - Handle WebSocket connection
// manages Chat, Moves, Ready Messages, and Game State
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
	// looked up by using the lobby ID as the map key
	currentLobby, exists := models.Lobbies[lobbyID]

	// if lobby exists, init a new state with an empty gameboard
	if exists {
		// Check if ChatMessages is already initialized, otherwise initialize as empty array
		if currentLobby.ChatMessages == nil {
			currentLobby.ChatMessages = []models.ChatMessage{}
		}
		// Check if ReadyPlayers is already initialized, otherwise initialize as empty map
		if currentLobby.ReadyPlayers == nil {
			currentLobby.ReadyPlayers = make(map[string]bool)
		}
	}

	// Proceed with assigning players to the lobby
	playerID := uuid.New().String()
	// A new player object is created with this unique ID, using Player struct in models.go
	player := &models.Player{ID: playerID}

	// Add the WebSocket connection to the lobby
	currentLobby.Conns = append(currentLobby.Conns, ws)

	//!TODO - investigate ways to simplify this into one function, avoid race
	// Send the current state to the newly connected client
	HandleInitialConnection(ws, currentLobby)

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

		switch msgType {
		case "setUsername":
			if username, ok := msg["userName"].(string); ok {
				// Now you can use the username as needed
				player.Name = username
				lobby.AssignAndNotifyPlayer(currentLobby, ws, player)
			} else {
				fmt.Println("Username is missing or not a string in the received message.")
			}
		case "chat":
			chat.HandleChatMessage(currentLobby, msg)
		case "move":
			// Extract the move details
			rawPosition := msg["position"].(float64)
			username := msg["userName"].(string)
			position := int(rawPosition)
			symbol := msg["symbol"].(string)

			response := currentLobby.Game.HandleGameMove(position, symbol, username)
			fmt.Println("resp", response)

			broadcastMove(currentLobby, response)

			for _, conn := range currentLobby.Conns {
				if err := websocket.JSON.Send(conn, response); err != nil {
					log.Printf("Error sending move message: %v", err)
					continue
				}
			}
		case "ready":
			// Mark the player as ready
			// add player to ReadyPlayers map
			// TODO!- add support for unready and check bool values
			currentLobby.ReadyPlayers[player.ID] = true

			// Check if both players are ready to start the game
			if len(currentLobby.ReadyPlayers) == 2 {

				// Broadcast startGame message to all connected clients
				start := map[string]interface{}{
					"type": "startGame",
				}

				// send statGame message to all Conns in the lobby
				for _, conn := range currentLobby.Conns {
					sendJSON(conn, start)
				}

				// Notify both players that the game is ready to start
				gameMasterMessage := map[string]interface{}{
					"type":   "chat",
					"sender": "GAMEMASTER",
					"text":   "Both players are ready. The game will start now!",
				}
				chat.HandleChatMessage(currentLobby, gameMasterMessage)
			}
		case "unready":
			fmt.Println("player unready: ", player)
			delete(currentLobby.ReadyPlayers, player.ID)
		}

	}
}

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

func broadcastMove(lobby *models.Lobby, result game.GameMessage) {
	switch result.Next {
	case "win":
		gameMasterMessage := map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   fmt.Sprintf("%v Wins!!", result.Winner),
		}
		chat.HandleChatMessage(lobby, gameMasterMessage)
	case "draw":
		gameMasterMessage := map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   "Its a Draw! Try Again!",
		}
		chat.HandleChatMessage(lobby, gameMasterMessage)
	}
}
