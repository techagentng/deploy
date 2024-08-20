package models

import "time"

type Bookmark struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null"`
	ReportID  string `gorm:"not null"` 
	CreatedAt time.Time
}