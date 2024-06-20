package models

// LGA struct
type LGA struct {
	ID       string `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	ReportID string `json:"report_id" gorm:"primaryKey"`
}

type State struct {
	ID       string `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	ReportID string `json:"report_id" gorm:"primaryKey"`
}
