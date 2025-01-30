package chat

import (
	"time"

	"tictacgo/models"

	"golang.org/x/net/websocket"
)

type Message struct {
	Text   string
	Sender string
}

func HandleChatMessage(conn *websocket.Conn, l *models.Lobby, player *models.Player, msg Message) error {
	// Convert msg to chat.Message struct
	chatMsg := Message{
		Text:   msg.Text,
		Sender: player.Name, // Assuming sender name is available from the player object
	}

	// Add message to lobby's chat history
	l.ChatMessages = append(l.ChatMessages, models.ChatMessage{
		Text:      chatMsg.Text,
		Sender:    chatMsg.Sender,
		Timestamp: time.Now(),
	})

	// Broadcast the updated chat messages
	return l.BroadcastChatMessages()
}
