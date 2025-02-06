package models

import (
	"tictacgo/pkg/game"
	"time"
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
	Game         *game.Game
	ReadyPlayers map[string]bool
	GameStarted  bool
	ChatMessages []ChatMessage
	CurrentTurn  string
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

type ChatMessage struct {
	Text      string
	Sender    string
	Timestamp time.Time
}
