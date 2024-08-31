package models

import "github.com/google/uuid"
// Reward represents rewards earned by users
type Role struct {
    ID   uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
    Name string    `json:"name"`
}

const (
    RoleUser  = "User"
    RoleAdmin = "Admin"
)
