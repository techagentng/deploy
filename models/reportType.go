package models

import "time"

// LGA struct
type ReportType struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"foreignKey:ID"`
	ReportID        string    `json:"report_id" gorm:"foreignKey:ID"`
	Category            string    `json:"category" binding:"required"`
	StateName       string    `json:"state_name" gorm:"foreignKey:Name" binding:"required"`
	LGAName         string    `json:"lga_name" gorm:"foreignKey:Name"`
	DateOfIncidence time.Time `json:"date_of_incidence" gorm:"column:date_of_incidence;foreignKey:DateOfIncidence"`
}

type ReportCriteria struct {
    ReportTypeCategory string     `json:"report_type_category"`
    States             []string   `json:"states"`
    StartDate          *time.Time `json:"start_date"`
    EndDate            *time.Time `json:"end_date"`
}

// SubReport represents subtypes of incident reports
type SubReport struct {
	ID           string `json:"id" gorm:"primaryKey;index"`
	ReportTypeID string `json:"report_type_id" gorm:"foreignKey:ID"`
	LGAID        string `json:"lga_id" gorm:"foreignKey:ID"`
	UserID       uint   `json:"user_id" gorm:"foreignKey:ID"`
	Name         string `json:"name" gorm:"not null"`
}

type StateReportCount struct {
	StateName   string `json:"state_name"`
	ReportCount int    `json:"report_count"`
}
