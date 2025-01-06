package ws

import (
	"fmt"
	"tictacgo/pkg/game"
	"tictacgo/pkg/types"

	"golang.org/x/net/websocket"
)

var Clients = make(map[*websocket.Conn]string)
var Spectators = make(map[*websocket.Conn]string)

func HandleConnections(ws *websocket.Conn, game *game.Game) {
	defer ws.Close()
	var userName string

	// Register user or spectator
	if game.UserCount >= 2 {
		game.SpectatorCount++
		userName = fmt.Sprintf("spectator-%d", game.SpectatorCount)
		Spectators[ws] = userName
		sendMessage(ws, types.Message{Type: "lobbyFull", Text: "Lobby full. You are a spectator.", UserName: userName})
		sendSystemMessage(fmt.Sprintf("%s joined as a spectator.", userName))
	} else {
		game.UserCount++
		userName = fmt.Sprintf("player-%d", game.UserCount)
		Clients[ws] = userName

		symbol := "X"
		if game.UserCount == 2 {
			symbol = "O"
		}

		sendMessage(ws, types.Message{Type: "assignPlayer", UserName: userName, Symbol: symbol})
		sendSystemMessage(fmt.Sprintf("%s joined the game.", userName))

		if game.UserCount == 2 && !game.GameStarted {
			game.GameStarted = true
			sendSystemMessage("Game started! X's turn.")
			sendMessageToAll(types.Message{Type: "updateTurn", Text: "X"})
		}
	}

	sendMessage(ws, types.Message{Type: "updateBoard", Text: fmt.Sprintf("%v", game.Board)})

	for {
		var msg types.Message
		err := websocket.JSON.Receive(ws, &msg)
		if err != nil {
			fmt.Println("Connection closed:", err)
			delete(Clients, ws)
			break
		}

		switch msg.Type {
		case "chat":
			sendMessageToAll(msg)
		case "move":
			handleMove(ws, msg.Position, msg.Symbol, game)
		}
	}
}

func handleMove(ws *websocket.Conn, position int, symbol string, game *game.Game) {
	if !game.MakeMove(position, symbol) || game.CurrentTurn != symbol {
		return
	}

	sendMessageToAll(types.Message{Type: "move", Position: position, Text: symbol})

	if winPatterns := game.CheckWin(symbol); len(winPatterns) > 0 {
		sendMessageToAll(types.Message{Type: "gameOver", Text: fmt.Sprintf("%s Wins!", symbol)})
		game.Reset()
		return
	}

	if game.CheckStalemate() {
		sendMessageToAll(types.Message{Type: "gameOver", Text: "It's a draw!"})
		game.Reset()
		return
	}

	game.SwitchTurn()
	sendMessageToAll(types.Message{Type: "updateTurn", Text: game.CurrentTurn})
}

func sendMessage(ws *websocket.Conn, msg types.Message) {
	websocket.JSON.Send(ws, msg)
}

func sendMessageToAll(msg types.Message) {
	for client := range Clients {
		sendMessage(client, msg)
	}
	for spectator := range Spectators {
		sendMessage(spectator, msg)
	}
}

func sendSystemMessage(text string) {
	sendMessageToAll(types.Message{Type: "system", Text: text})
}
