package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	ConversationID uuid.UUID   `gorm:"type:uuid;not null" json:"conversation_id"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID" json:"conversation"`
	SenderID       uuid.UUID   `gorm:"type:uuid;not null" json:"sender_id"`
	Sender         User        `gorm:"foreignKey:SenderID" json:"sender"`
	Content        string      `json:"content"`
	CreatedAt      time.Time   `json:"created_at"`
	IsRead         bool        `gorm:"default:false" json:"is_read"`
}
