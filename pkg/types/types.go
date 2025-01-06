package types

type Message struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Sender   string `json:"sender,omitempty"`
	UserName string `json:"userName,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Position int    `json:"position"` // Allow zero int value
}
