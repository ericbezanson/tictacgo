package chat

import (
	"encoding/json"
	"fmt"
	"log"
	"tictacgo/models"

	"time"

	"golang.org/x/net/websocket"
)

type Message struct {
	Text   string
	Sender string
}

func HandleChatMessage(lobbyID string, msg map[string]interface{}, connections map[string][]*websocket.Conn) error {

	sender, senderOk := msg["sender"].(string)
	text, textOk := msg["text"].(string)

	println("sender", sender)
	println("text", text)

	if !senderOk || !textOk {
		log.Printf("Invalid chat message: %+v", msg)
		return nil // Prevent the app from crashing
	}

	chatMsg := Message{
		Text:   msg["text"].(string),
		Sender: msg["sender"].(string),
	}

	// Find the lobby in models
	l, exists := models.Lobbies[lobbyID]
	if !exists {
		return fmt.Errorf("lobby not found")
	}

	// Add message to lobby's chat history
	l.ChatMessages = append(l.ChatMessages, models.ChatMessage{
		Text:      chatMsg.Text,
		Sender:    chatMsg.Sender,
		Timestamp: time.Now(),
	})

	// Broadcast the updated chat messages
	return BroadcastChatMessages(lobbyID, l.ChatMessages, connections)

}

func BroadcastChatMessages(lobbyID string, messages []models.ChatMessage, connections map[string][]*websocket.Conn) error {

	msg := struct {
		Type         string               `json:"type"`
		ChatMessages []models.ChatMessage `json:"chatMessages"`
	}{
		Type:         "chat",
		ChatMessages: messages,
	}

	// Send message to all clients in the lobby
	for _, conn := range connections[lobbyID] {

		err := json.NewEncoder(conn).Encode(msg)
		if err != nil {
			fmt.Printf("Error sending chat message: %v\n", err)
			continue
		}
	}

	return nil
}
