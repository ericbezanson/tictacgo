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
	Ready  bool
}

type Lobby struct {
	ID           string
	Name         string
	MaxPlayers   int
	Players      []*Player
	Conns        []*websocket.Conn // Track active WebSocket connections
	Game         *game.Game
	GameBoard    [9]string       `json:"gameBoard"`
	CurrentTurn  string          `json:"currentTurn"`
	GameStarted  bool            `json:"gameStarted"`
	ChatMessages []Message       `json:"chatMessages"`
	ReadyPlayers map[string]bool `json:"readyPlayers"`
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

type Game struct {
	Board          [9]string
	CurrentTurn    string
	GameStarted    bool
	UserCount      int
	SpectatorCount int
	Players        []string // track player names/symbols
}
