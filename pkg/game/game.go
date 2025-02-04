package game

import (
	"fmt"
)

type Game struct {
	Board          [9]string
	CurrentTurn    string
	GameStarted    bool
	UserCount      int
	SpectatorCount int
	Players        []string // track player names/symbols
}

type GameMessage struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Next     string `json:"next"`
	Winner   string `json:"winner"`
	Position int    `json:"position"`
	Symbol   string `json:"symbol"`
}

// --------------------------------------------------------------------------------- GAME / SERVER COMMUNICATION

// HandleGameMove processes the move and returns a GameMessage
func (g *Game) HandleGameMove(position int, symbol string, username string) GameMessage {
	if position < 0 || position > 8 || g.Board[position] != "" {
		return GameMessage{
			Type: "invalidMove",
			Text: "Invalid move: Position already filled or out of bounds",
		}
	}

	g.Board[position] = symbol

	if winPatterns := g.CheckWin(symbol); len(winPatterns) > 0 {
		g.Reset()
		return GameMessage{
			Type:     "move",
			Text:     fmt.Sprintf("%s wins!", username),
			Next:     "win",
			Winner:   symbol,
			Position: position,
			Symbol:   symbol,
		}
	} else if g.CheckStalemate() {
		g.Reset()
		return GameMessage{
			Type:     "move",
			Text:     "It's a draw!",
			Next:     "draw",
			Winner:   "none",
			Position: position,
			Symbol:   symbol,
		}
	} else {
		g.SwitchTurn()
		return GameMessage{
			Type:     "move",
			Text:     g.CurrentTurn,
			Next:     "updateTurn",
			Winner:   "none",
			Position: position,
			Symbol:   symbol,
		}
	}
}

// / -------------------------------------------------------------------------------- GAME LOGIC

// inits new instance of Game, with default values
func NewGame() *Game {
	return &Game{
		Board:       [9]string{"", "", "", "", "", "", "", "", ""},
		CurrentTurn: "X", // X always starts
		GameStarted: false,
		UserCount:   0,
		Players:     []string{},
	}
}

// Marks the game as started by setting GameStarted to true.
func (g *Game) Start() {
	g.GameStarted = true
}

// Handles player moves on the game board.
func (g *Game) MakeMove(position int, symbol string) bool {
	if position < 0 || position > 8 || g.Board[position] != "" {
		return false
	}
	g.Board[position] = symbol
	return true
}

// Alternates the game turn between "X" and "O".
func (g *Game) SwitchTurn() {
	if g.CurrentTurn == "X" {
		g.CurrentTurn = "O"
	} else {
		g.CurrentTurn = "X"
	}
}

// check for wins against win patterns
func (g *Game) CheckWin(symbol string) [][3]int {
	winPatterns := [][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
		{0, 4, 8}, {2, 4, 6},
	}

	var winningPatterns [][3]int
	for _, pattern := range winPatterns {
		if g.Board[pattern[0]] == symbol && g.Board[pattern[1]] == symbol && g.Board[pattern[2]] == symbol {
			winningPatterns = append(winningPatterns, pattern)
		}
	}
	return winningPatterns
}

// check for stalemate, aka all tiles filled with no win
func (g *Game) CheckStalemate() bool {
	for _, cell := range g.Board {
		if cell == "" {
			return false
		}
	}
	return true
}

// reset game after win or draw
func (g *Game) Reset() {
	g.Board = [9]string{"", "", "", "", "", "", "", "", ""}
	g.CurrentTurn = "X"
	g.GameStarted = false
}
