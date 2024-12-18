package game

type Game struct {
	Board          [9]string
	CurrentTurn    string
	GameStarted    bool
	UserCount      int
	SpectatorCount int
}

func NewGame() *Game {
	return &Game{
		Board:       [9]string{"", "", "", "", "", "", "", "", ""},
		CurrentTurn: "X",
		GameStarted: false,
		UserCount:   0,
	}
}

func (g *Game) Reset() {
	g.Board = [9]string{"", "", "", "", "", "", "", "", ""}
	g.CurrentTurn = "X"
	g.GameStarted = false
	g.UserCount = 0
}

func (g *Game) MakeMove(position int, symbol string) bool {
	if position < 0 || position > 8 || g.Board[position] != "" {
		return false
	}
	g.Board[position] = symbol
	return true
}

func (g *Game) SwitchTurn() {
	if g.CurrentTurn == "X" {
		g.CurrentTurn = "O"
	} else {
		g.CurrentTurn = "X"
	}
}

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

func (g *Game) CheckStalemate() bool {
	for _, cell := range g.Board {
		if cell == "" {
			return false
		}
	}
	return true
}
