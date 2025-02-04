package models

import (
	"encoding/json"
	"tictacgo/pkg/game"
	"time"

	"golang.org/x/net/websocket"
)

// A global map storing active game lobbies.
var Lobbies = make(map[string]*Lobby)

// NOTE: Go’s structs are typed collections of fields. They’re useful for grouping data together to form records.

type Player struct {
	ID     string
	Symbol string
	Name   string
	Ready  bool
}

type Lobby struct {
	ID           string
	Name         string
	MaxPlayers   int
	Players      []*Player
	Conns        []*websocket.Conn
	Game         *game.Game // ✅ Reference Game directly
	ReadyPlayers map[string]bool
	GameStarted  bool          `json:"gameStarted"`
	ChatMessages []ChatMessage `json:"chatMessages"`
	CurrentTurn  string        `json:"currentTurn"`
}

func (l *Lobby) BroadcastChatMessages() error {
	// Create a message to send to clients
	msg := struct {
		Type         string        `json:"type"`
		ChatMessages []ChatMessage `json:"chatMessages"`
	}{
		Type:         "updatePlayers",
		ChatMessages: l.ChatMessages,
	}

	// Send the message to all connected clients
	for _, conn := range l.Conns {
		err := json.NewEncoder(conn).Encode(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

type Message struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Sender    string `json:"sender,omitempty"`
	UserName  string `json:"userName,omitempty"`
	Symbol    string `json:"symbol,omitempty"`
	Position  int    `json:"position"`
	Timestamp string `json:"timestamp"`
}

// type Game struct {
// 	Board          [9]string
// 	CurrentTurn    string
// 	GameStarted    bool
// 	UserCount      int
// 	SpectatorCount int
// 	Players        []string // track player names/symbols
// }

type ChatMessage struct {
	Text      string
	Sender    string
	Timestamp time.Time
}
