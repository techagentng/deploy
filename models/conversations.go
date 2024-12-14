package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
    ID           uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"` // Unique identifier
    Participants []User     `gorm:"many2many:conversation_participants;" json:"participants"` // Many-to-many relation with users
    LastMessage  string     `json:"last_message"`   // Last message sent in the conversation
    UpdatedAt    time.Time  `json:"updated_at"`     // Last update timestamp
    CreatedAt    time.Time  `json:"created_at"`     // Creation timestamp
}
