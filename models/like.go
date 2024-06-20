package models

// Like represents a user's like on an incident report
type Like struct {
	ID     string `json:"id" gorm:"primaryKey"`
	UserID uint   `json:"user_id" gorm:"foreignKey:ID"`
	// IncidentReportID string `json:"report_id" gorm:"foreignKey:ID"`
	Count int `json:"count"`
}

type View struct {
	ID               string `gorm:"primaryKey"`
	IncidentReportID string `gorm:"foreignKey:ID"`
}
