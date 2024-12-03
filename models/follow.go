package models

import (
	"time"

	"github.com/google/uuid"
)

type Follow struct {
    ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
    UserID      uint      `json:"user_id" gorm:"not null"`
    ReportID    uuid.UUID `json:"report_id" gorm:"not null"`
    FollowText  string    `json:"follow_text" gorm:"type:text"` // Required field
    FollowMedia string    `json:"follow_media" gorm:"type:text"`         // Optional field
    CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}
