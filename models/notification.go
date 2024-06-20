package models

// Notification represents notifications sent to users
type Notification struct {
	Model
	UserID  uint   `json:"user_id" gorm:"foreignKey:UserID"`
	Message string `json:"message"`
	IsRead  bool   `json:"is_read"`
}
