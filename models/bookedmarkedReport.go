package models

import (
	"github.com/google/uuid"
	"time"
)

type Bookmark struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	ReportID  uuid.UUID `gorm:"not null"`
	CreatedAt time.Time
}
