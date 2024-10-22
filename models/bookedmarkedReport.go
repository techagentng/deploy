package models

import (
	"time"
	"github.com/google/uuid"
)

type Bookmark struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null"`
	ReportID  uuid.UUID `gorm:"not null"`
	CreatedAt time.Time
}
