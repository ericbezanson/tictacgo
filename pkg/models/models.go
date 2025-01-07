package models

import (
	"tictacgo/pkg/game"

	"golang.org/x/net/websocket"
)

// A global map storing active game lobbies.
var Lobbies = make(map[string]*Lobby)

// NOTE: Go’s structs are typed collections of fields. They’re useful for grouping data together to form records.

type Player struct {
	ID     string
	Symbol string
	Name   string
}

type Lobby struct {
	ID         string
	Name       string
	MaxPlayers int
	Players    []*Player
	Conns      []*websocket.Conn // Add this field to track active WebSocket connections
	Game       *game.Game
}

type Message struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Sender   string `json:"sender,omitempty"`
	UserName string `json:"userName,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Position int    `json:"position"`
}

type Game struct {
	Board          [9]string
	CurrentTurn    string
	GameStarted    bool
	UserCount      int
	SpectatorCount int
	Players        []string // track player names/symbols
}
