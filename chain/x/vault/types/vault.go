package types

import (
	"time"
)

// Message represents a stored message with metadata
type Message struct {
	Sender    string    `json:"sender"`    // Cosmos bech32 address
	Content   string    `json:"content"`   // Message content
	Timestamp time.Time `json:"timestamp"` // When message was stored
	Index     uint64    `json:"index"`     // User's message index
}

// NewMessage creates a new Message
func NewMessage(sender, content string, timestamp time.Time, index uint64) Message {
	return Message{
		Sender:    sender,
		Content:   content,
		Timestamp: timestamp,
		Index:     index,
	}
}
