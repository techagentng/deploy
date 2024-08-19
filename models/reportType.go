package models

import "time"

// LGA struct
type ReportType struct {
	ID                   string     `json:"id" gorm:"primaryKey"`
	UserID               uint       `json:"user_id"`  
	ReportID             string     `json:"report_id"`  
	Category             string     `json:"category" binding:"required"`
	StateName            string     `json:"state_name"`
	LGAName              string     `json:"lga_name"`
	IncidentReportRating string     `json:"incident_report_rating"`
	DateOfIncidence      time.Time  `json:"date_of_incidence"`
	SubReports           []SubReport `gorm:"foreignKey:ReportTypeID"` 
}

type SubReport struct {
	ID             string `json:"id" gorm:"primaryKey;index"`
	ReportTypeID   string `json:"report_type_id"`  // Foreign key referring to ReportType table
	SubReportType  string `json:"sub_report_type"`
}

type RatingPercentage struct {
	GoodPercentage float64 `json:"good_percentage"`
	BadPercentage  float64 `json:"bad_percentage"`
}
type ReportCriteria struct {
	ReportTypes []string   `json:"report_types"`
	States      []string   `json:"states"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}

type StateReportCount struct {
	StateName   string `json:"state_name"`
	Category    string `json:"category"`
	ReportCount int    `json:"report_count"`
}
