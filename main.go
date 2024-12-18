package main

import (
	"fmt"
	"net/http"

	"golang.org/x/net/websocket" // switched from gorilla
)

// Message Struct for data being sent over websocket
// Type: Describes the type of message, such as a chat message or a move in the game.
// Text: The content of the message (e.g., "User X has joined the game").
// Sender: An optional field for the sender's name.
// UserName: An optional field for the user's unique name (e.g., user-1).
// Symbol: An optional field for the player's symbol, either "X" or "O".
// Position: A pointer to an integer, representing the position on the Tic-Tac-Toe board (optional and can be nil).
type Message struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Sender   string `json:"sender,omitempty"`
	UserName string `json:"userName,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Position int    `json:"position"` // Allow for zero int value
}

// Globals

// Go map: data structure that acts as a collection of unordered key-value pairs
// use map here because we needed to store a key value pair (*websocket.Conn as a unique key) of a dynamic size
// NOTE: use the memory address as a unique key in the map
var clients = make(map[*websocket.Conn]string)

// same as clients, however this will store connection pointers for players who are not able to interact with game board
var spectators = make(map[*websocket.Conn]string)

// spectator count - used in spectator naming
var spectatorCount int

// A fixed-size array of strings representing the Tic-Tac-Toe board. Each element can be empty (""), "X", or "O".
// NOTE: more effecient to use a fixed sized array as TTT board will always be 3x3
var board = [9]string{"", "", "", "", "", "", "", "", ""}

// Track the number of users connected to the game.
var userCount int

// track if game is started, (needs to players)
var gameStarted bool

// keeps track of current player, default X for player one
var currentPlayer = "X"

func main() {
	// Sets up a handler to serve static files from current dir ./
	http.Handle("/", http.FileServer(http.Dir("./")))

	// sets up a WebSocket handler at the /ws path.
	http.Handle("/ws", websocket.Handler(handleConnections))

	// Starts the HTTP server on port 8080
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Server Error:", err)
	}
}

func handleConnections(ws *websocket.Conn) {
	// defers the execution until the surrounding function returns.
	defer ws.Close()

	// username bucket, can be a user or spectator
	var userName string

	// Register user
	// if more than 2 users, spectator role assigned
	if userCount >= 2 {
		spectatorCount++
		userName = fmt.Sprintf("spectator-%d", spectatorCount)
		clients[ws] = userName

		// Notify spectator of status
		sendMessage(ws, Message{
			Type:     "lobbyFull",
			Text:     "The game lobby is full. You are now spectating.",
			UserName: userName,
		})

		// Broadcast spectator join message
		sendSystemMessage(fmt.Sprintf("%s has joined as a spectator.", userName))
	} else {
		// if not a spectator, assign user role
		userCount++
		userName = fmt.Sprintf("player-%d", userCount)
		clients[ws] = userName

		// Assign player symbol
		assignSymbol := "X"
		if userCount == 2 {
			assignSymbol = "O"
		}

		// Notify player of assignment
		sendMessage(ws, Message{
			Type:     "assignPlayer",
			UserName: userName,
			Symbol:   assignSymbol,
		})

		// Broadcast player join message
		sendSystemMessage(fmt.Sprintf("%s has joined the game.", userName))

		// Start the game when two players have joined
		if userCount == 2 && !gameStarted {
			gameStarted = true
			sendSystemMessage("Game has started! It's X's turn.")
			sendMessageToAll(Message{Type: "updateTurn", Text: "X"})
		}
	}

	// Send the initial board state to the new user
	sendMessage(ws, Message{
		Type: "updateBoard",
		Text: fmt.Sprintf("%v", board),
	})

	// Listen for messages
	for {
		var msg Message
		err := websocket.JSON.Receive(ws, &msg)
		if err != nil {
			fmt.Println("Connection closed:", err)
			delete(clients, ws)
			break
		}

		// Handle chat or move messages
		switch msg.Type {
		case "chat":
			sendMessageToAll(msg)
		case "move":
			handleMove(ws, msg.Position, msg.UserName, msg.Symbol)
		}
	}
}

// args
// ws *websocket.Conn: The WebSocket connection of the player making the move. (pointer)
// position *int: A pointer to the board position where the player wants to place their symbol (accept 0 value).
// symbol string: The player’s symbol ("X" or "O").
func handleMove(ws *websocket.Conn, position int, sender string, symbol string) {
	// Validate the move
	if position < 0 || position > 8 {
		fmt.Println("Invalid move: Position is out of bounds")
		return
	}
	if currentPlayer != symbol {
		fmt.Println("Invalid move: Not your turn")
		return
	}
	if board[position] != "" {
		fmt.Println("Invalid move: Cell already occupied")
		return
	}

	// Update the game board / place symbol in clicked tile
	board[position] = symbol

	// Broadcast the move to all clients
	sendMessageToAll(Message{
		Type:     "move",
		Position: position,
		Text:     symbol,
	})

	// Check if the current move resulted in a win
	if winPattern := checkWin(symbol); len(winPattern) > 0 {
		// Announce the winner
		sendMessageToAll(Message{
			Type:     "gameOver",
			Text:     fmt.Sprintf("User-%s Wins!", symbol),
			Symbol:   symbol,
			Position: -1, // Unused
		})

		// Reset the game
		resetGame()
		return
	}

	// If no win, check for a draw
	if checkStalemate() {
		sendMessageToAll(Message{
			Type: "gameOver",
			Text: "It's a draw!",
		})
		resetGame()
		return
	}

	// Switch turns
	switchTurn()

	// Notify players of the turn change
	sendMessageToAll(Message{Type: "updateTurn", Text: currentPlayer})
}

// Check if the given symbol has won
func checkWin(symbol string) [][3]int {
	// all possible win patterns in tic tac toe
	// slice of arrays [3]int
	winPatterns := [][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // Rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // Columns
		{0, 4, 8}, {2, 4, 6}, // Diagonals
	}

	// iterate over all win patterns and check if the player has won
	var winningPatterns [][3]int
	for _, pattern := range winPatterns {
		if board[pattern[0]] == symbol && board[pattern[1]] == symbol && board[pattern[2]] == symbol {
			winningPatterns = append(winningPatterns, pattern)
		}
	}
	return winningPatterns
}

func checkStalemate() bool {
	for _, cell := range board {
		if cell == "" {
			return false // There's still an empty cell
		}
	}
	return true
}

func resetGame() {
	// Reset the board and game state
	board = [9]string{"", "", "", "", "", "", "", "", ""}
	gameStarted = false
	userCount = 0
	currentPlayer = "X"
}

func switchTurn() {
	if currentPlayer == "X" {
		currentPlayer = "O"
	} else {
		currentPlayer = "X"
	}
}

func sendMessage(ws *websocket.Conn, msg Message) {
	websocket.JSON.Send(ws, msg)
}

func sendMessageToAll(msg Message) {
	fmt.Printf("Broadcasting message: %+v\n", msg)
	for client := range clients {
		sendMessage(client, msg)
	}
	for spectator := range spectators {
		sendMessage(spectator, msg)
	}
}

func sendSystemMessage(text string) {
	sendMessageToAll(Message{Type: "system", Text: text})
}

// TODO
// ------ MAJOR ------
// - Add start button / player ready
// - allow spectator to see board state if joining midgame
// - graceful shut down
// - update hardcoded localhost
// - isolate into new files (types)
// - add custom names
// - BUG: player 1 can interact with game board before game starts
// -----
// - broadcast state
// - keep gamestate checks on server side
// - multi games
// - lobby system
// - unit test, check win
// - table driven tests

// ------ MINOR ------
// highlight winning pattern
// chat with enter button
