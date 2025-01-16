package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"tictacgo/pkg/models"
	"tictacgo/pkg/redis"

	"golang.org/x/net/websocket"
)

// globals
// store active game lobbies in a map, a map in go is used to store key value pairs, dictionary-like
var Lobbies = make(map[string]*models.Lobby)
var ctx = context.Background()

// SaveLobbyState saves the lobby state to Redis
func SaveLobbyState(lobbyID string, lobby *models.Lobby) error {
	// debugLobbyData, err := json.MarshalIndent(models.Lobbies[lobbyID], "", "  ")
	// if err != nil {
	// 	fmt.Println("Error marshalling lobby data:", err)
	// } else {
	// 	fmt.Println("Lobby details from REDIS:", string(debugLobbyData))
	// }
	stateJSON, err := json.Marshal(models.Lobbies[lobbyID])
	if err != nil {
		return err
	}

	err = redis.Client.Set(ctx, "lobby:"+lobbyID, stateJSON, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

// LoadLobbyState loads the lobby state from Redis
func LoadLobbyState(lobbyID string) (*models.LobbyState, error) {
	stateJSON, err := redis.Client.Get(ctx, "lobby:"+lobbyID).Result()
	// debugLobbyData, err := json.MarshalIndent(models.Lobbies[lobbyID], "", "  ")

	// if err != nil {
	// 	fmt.Println("Error marshalling lobby data:", err)
	// } else {
	// 	fmt.Println("Lobby details from REDIS:", string(debugLobbyData))
	// }
	// fmt.Println("stateJSON", stateJSON)
	if err != nil {
		return nil, err
	}

	var lobbyState models.LobbyState
	err = json.Unmarshal([]byte(stateJSON), &lobbyState)
	if err != nil {
		return nil, err
	}

	return &lobbyState, nil
}

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
	lobby, exists := models.Lobbies[lobbyID]

	// if lobby exists but state is nil, init a new state with an empty gameboard
	if exists && lobby.State == nil {
		lobby.State = &models.LobbyState{
			GameBoard:    [9]string{},
			ChatMessages: []models.Message{},
			Players:      []models.Player{},
			ReadyPlayers: map[string]bool{}, // Track readiness of players
		}
	}

	// Log the check result for debugging
	fmt.Printf("Lobby exists: %v Lobby data: %+v\n", exists, lobby)

	// Send the current state to the newly connected client
	HandleInitialConnection(ws, lobby)

	// // Broadcast the updated lobby state
	// BroadcastLobbyState(lobby)

	// Continuously handle incoming messages
	lobby.Conns = append(lobby.Conns, ws)
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
		case "open":

			// Extract playerID from the message
			playerID, ok := msg["playerID"].(string)
			if !ok {
				fmt.Println("Error: PlayerID not found in open message")
				continue
			}

			// Check if the playerID already exists in the Players array
			foundExistingPlayer := false
			for _, existingPlayer := range lobby.State.Players { // Iterate over models.Player objects
				if existingPlayer.ID == playerID { // Compare player IDs
					// Prepare and send the message to individual player
					gameMasterMsg := map[string]interface{}{
						"type":     "assignPlayer",
						"userName": existingPlayer.Name,
						"symbol":   existingPlayer.Symbol,
					}
					sendJSON(ws, gameMasterMsg)

					foundExistingPlayer = true
					// Broadcast lobby state and notify other players
					BroadcastLobbyState(lobby)

					BroadcastChatMessage(lobby, map[string]interface{}{
						"type":   "chat",
						"sender": "GAMEMASTER",
						"text":   fmt.Sprintf("welcome back %s", existingPlayer.Name),
					})

					break
				}
			}

			// If playerID exists, reuse the existing player object
			if foundExistingPlayer {
				fmt.Printf("Player %s reconnected\n", playerID)

				continue // No need to re-assign player symbol or name
			}

			// If playerID doesn't exist, proceed with assigning players
			player := &models.Player{ID: playerID}

			if len(lobby.State.Players) < 2 {
				// Assign the first two players as "X" and "O"
				symbol := "X"
				if len(lobby.State.Players) == 1 {
					symbol = "O"
				}
				player.Symbol = symbol
				player.Name = fmt.Sprintf("%s (%s)", playerID, symbol)
				lobby.State.Players = append(lobby.State.Players, *player)

				// Prepare and send the message to individual player
				gameMasterMsg := map[string]interface{}{
					"type":     "assignPlayer",
					"userName": player.Name,
					"symbol":   player.Symbol,
				}
				sendJSON(ws, gameMasterMsg)

				// Notify all about the new player joining
				BroadcastChatMessage(lobby, map[string]interface{}{
					"type":   "chat",
					"sender": "GAMEMASTER",
					"text":   fmt.Sprintf("%s has joined the game!, playing as %s", player.Name, symbol),
				})

				// Broadcast lobby state and notify other players
				BroadcastLobbyState(lobby)

			} else {
				// Assign additional connections as spectators
				player.Symbol = "S" // Spectator symbol
				player.Name = fmt.Sprintf("%v (Spectator)", playerID)
				lobby.State.Players = append(lobby.State.Players, *player)

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
					"text":   fmt.Sprintf("%v is now spectating!", player.Name),
				})

				// Broadcast lobby state and notify other players
				BroadcastLobbyState(lobby)
			}

		case "chat":
			BroadcastChatMessage(lobby, msg)
		case "move":
			BroadcastGameMove(lobby, ws, msg)
		case "ready":
			// Mark the player as ready
			// add player to ReadyPlayers map
			// TODO!- add support for unready and check bool values
			// lobby.State.ReadyPlayers[player.ID] = true

			// // Check if both players are ready to start the game
			// if len(lobby.State.ReadyPlayers) == 2 {

			// 	// Broadcast startGame message to all connected clients
			// 	startGameMsg := map[string]interface{}{
			// 		"type": "startGame",
			// 	}

			// 	// send statGame message to all Conns in the lobby
			// 	for _, conn := range lobby.Conns {
			// 		sendJSON(conn, startGameMsg)
			// 	}

			// 	// Notify both players that the game is ready to start
			// 	BroadcastChatMessage(lobby, map[string]interface{}{
			// 		"type":   "startGame",
			// 		"sender": "GAMEMASTER",
			// 		"text":   "Both players are ready. The game will start now!",
			// 	})
			// }
		}
	}
}

////// ------------------------------------------------------------------------------------------------------------- Helpers

// broadcast state to a newly connected user when they first connect to the lobby
func HandleInitialConnection(ws *websocket.Conn, lobby *models.Lobby) {

	severLobbyState, err := LoadLobbyState(lobby.ID)

	if severLobbyState != nil {
		networkState := struct {
			Type  string             `json:"type"`
			State *models.LobbyState `json:"state"`
		}{
			Type:  "initialState",
			State: lobby.State,
		}
		websocket.JSON.Send(ws, networkState)
	}

	if err != nil {
		log.Println("Error loading lobbystate:", err)
		return
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

// updates all connected clients with the current lobby state

// updates all connected clients with the current lobby state
func BroadcastLobbyState(lobby *models.Lobby) {
	fmt.Println("BroadcastLobbyState", lobby)

	state := struct {
		Type  string             `json:"type"`
		State *models.LobbyState `json:"state"`
	}{
		Type:  "updatePlayers",
		State: lobby.State,
	}

	fmt.Println("Lobby Conns", lobby.Conns)
	// Send state to all connected clients
	for _, conn := range lobby.Conns {
		fmt.Println("broadcastlobbystate conn", conn)
		// Log the actual data stored in conn (if possible)
		connData, err := json.Marshal(conn)
		if err != nil {
			log.Printf("Error marshalling conn data: %v", err)
		} else {
			fmt.Println("CONN Data:", string(connData))
		}

		fmt.Println("STATE", state)
		if err := websocket.JSON.Send(conn, state); err != nil {
			fmt.Printf("Error sending state to connection: %v\n", err)
		}
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

	fmt.Println("broadcastchatmessage", lobby.Conns)
	for _, conn := range lobby.Conns {
		fmt.Println("CONN Loop", lobby.Conns)
		printConnDetails(conn)
		if err := websocket.JSON.Send(conn, msg); err != nil {
			fmt.Printf("Error broadcasting message to %v: %v\n", conn.Request().RemoteAddr, err)
		} else {
			fmt.Printf("Successfully sent message to %v: %+v\n", conn.Request().RemoteAddr, msg)
		}
	}

	// save to redis
	SaveLobbyState(lobby.ID, lobby)
}

func printConnDetails(conn *websocket.Conn) {
	if conn == nil {
		fmt.Println("conn is nil")
		return
	}
	fmt.Printf("RemoteAddr: %v\n", conn.Request().RemoteAddr)
	// Add other relevant fields here (e.g., Subprotocols, WriteChan, etc.)
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

		// save to redis
		SaveLobbyState(lobby.ID, lobby)
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
