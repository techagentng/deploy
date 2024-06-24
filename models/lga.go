package models

// LGA struct
type LGA struct {
	ID       string `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	ReportTypeID string `json:"report_type_id" gorm:"foreignKey:ID"`
}
type State struct {
	ID       string `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	ReportID string `json:"report_id" gorm:"primaryKey"`
}
