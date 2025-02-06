package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"tictacgo/models"
	"tictacgo/pkg/chat"
	"tictacgo/pkg/game"
	"tictacgo/pkg/lobby"

	"github.com/go-redis/redis"
	"golang.org/x/net/websocket"
)

// globals
// store active game lobbies in a map, a map in go is used to store key value pairs, dictionary-like
var LobbyConnections = make(map[string][]*websocket.Conn)

var redisClient = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

// HandleWebSocket - Handle WebSocket connection
// manages Chat, Moves, Ready Messages, and Game State
// Ensure only one active connection per player
func HandleWebSocket(ws *websocket.Conn) {
	query := ws.Request().URL.Query()
	lobbyID := query.Get("lobby")

	if lobbyID == "" {
		fmt.Println("Lobby ID is required")
		ws.Close()
		return
	}

	currentLobby, exists := models.Lobbies[lobbyID]
	if !exists {
		fmt.Println("Lobby not found")
		ws.Close()
		return
	}

	if _, exists := LobbyConnections[lobbyID]; !exists {
		LobbyConnections[lobbyID] = []*websocket.Conn{}
	}

	// Remove stale connections before adding a new one
	removeDuplicateConnection(lobbyID, ws)
	// Add the new connection
	LobbyConnections[lobbyID] = append(LobbyConnections[lobbyID], ws)

	// Handle connection cleanup on disconnect
	defer func() {
		fmt.Println("Client disconnected, removing from lobby")
		removeConnection(currentLobby.ID, ws)
	}()

	HandleInitialConnection(ws, currentLobby)

	// Handle incoming messages
	for {
		var msg map[string]interface{}
		err := websocket.JSON.Receive(ws, &msg)
		if err != nil {
			fmt.Printf("Error receiving message: %v\n", err)
			break
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "setUsername":
			username, ok := msg["username"].(string)
			if !ok {
				continue
			}
			var id string
			if idVal, ok := msg["id"].(string); ok {
				id = idVal
			}

			lobby.AssignAndNotifyPlayer(currentLobby, ws, username, id, LobbyConnections)

			storeLobbyState(lobbyID, currentLobby)
		case "chat":
			chat.HandleChatMessage(currentLobby.ID, msg, LobbyConnections)
			storeLobbyState(lobbyID, currentLobby)
		case "move":
			rawPosition := msg["position"].(float64)
			username := msg["username"].(string)
			position := int(rawPosition)
			symbol := msg["symbol"].(string)

			response := currentLobby.Game.HandleGameMove(position, symbol, username)
			broadcastMove(currentLobby, response)
			storeLobbyState(lobbyID, currentLobby)

			for _, conn := range LobbyConnections[currentLobby.ID] {
				if err := websocket.JSON.Send(conn, response); err != nil {
					log.Printf("Error sending move message: %v", err)
					continue
				}
			}
		case "ready":
			username := msg["username"].(string)
			ready := msg["ready"].(bool)

			if ready {
				currentLobby.ReadyPlayers[username] = true
				if len(currentLobby.ReadyPlayers) == 2 && !currentLobby.GameStarted {
					currentLobby.GameStarted = true // Prevent duplicate start messages
					start := map[string]interface{}{
						"type": "startGame",
					}
					for _, conn := range LobbyConnections[currentLobby.ID] {
						sendJSON(conn, start)
					}

					gameMasterMessage := map[string]interface{}{
						"type":   "chat",
						"sender": "GAMEMASTER",
						"text":   "Both players are ready. The game will start now!",
					}
					chat.HandleChatMessage(currentLobby.ID, gameMasterMessage, LobbyConnections)
				}
			} else if !ready {
				delete(currentLobby.ReadyPlayers, username)
				fmt.Printf("Player %s is no longer ready\n", username)
			}
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

		lobby.Game.Reset()
		lobby.GameStarted = false
		chat.HandleChatMessage(lobby.ID, gameMasterMessage, LobbyConnections)
	case "draw":
		gameMasterMessage := map[string]interface{}{
			"type":   "chat",
			"sender": "GAMEMASTER",
			"text":   "Its a Draw! Try Again!",
		}
		lobby.Game.Reset()
		lobby.GameStarted = false
		chat.HandleChatMessage(lobby.ID, gameMasterMessage, LobbyConnections)
	}
}

func storeLobbyState(lobbyID string, lobby *models.Lobby) {
	// Convert lobby state to JSON
	lobbyJSON, err := json.Marshal(lobby)
	if err != nil {
		log.Printf("Error marshalling lobby state: %v", err)
		return
	}

	// Store in Redis without expiration (manual deletion)
	err = redisClient.Set("lobby:"+lobbyID, lobbyJSON, 0).Err()
	if err != nil {
		log.Printf("Error storing lobby state in Redis: %v", err)
	} else {
		log.Printf("Lobby %s state saved in Redis", lobbyID)
	}
}

func removeDuplicateConnection(lobbyID string, newConn *websocket.Conn) {
	var activeConns []*websocket.Conn

	for _, conn := range LobbyConnections[lobbyID] {
		if conn == newConn {
			fmt.Println("Duplicate connection found, removing old one.")
			conn.Close() // Close the old connection
			continue
		}
		activeConns = append(activeConns, conn)
	}

	LobbyConnections[lobbyID] = activeConns
}

func removeConnection(lobbyID string, conn *websocket.Conn) {
	var activeConns []*websocket.Conn

	for _, c := range LobbyConnections[lobbyID] {
		if c == conn {
			fmt.Println("Closing stale connection")
			c.Close()
			continue
		}
		activeConns = append(activeConns, c)
	}

	LobbyConnections[lobbyID] = activeConns
}
