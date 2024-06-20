package models

type BookmarkReport struct {
	ID       string `json:"id" gorm:"primaryKey"`
	UserID   uint   `json:"user_id" gorm:"foreignKey:ID"`
	ReportID string `json:"report_id"`
}
