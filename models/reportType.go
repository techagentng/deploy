package models

import "time"

// LGA struct
type ReportType struct {
	ID                   string    `json:"id" gorm:"primaryKey"`
	UserID               uint      `json:"user_id" gorm:"foreignKey:UserID"` // ForeignKey should reference the field in the User table
	ReportID             string    `json:"report_id" gorm:"foreignKey:ReportID"` // ForeignKey should reference the field in the Report table
	Category             string    `json:"category" binding:"required"`
	StateName            string    `json:"state_name" gorm:"foreignKey:StateName"` // ForeignKey should reference the field in the State table
	LGAName              string    `json:"lga_name" gorm:"foreignKey:LGAName"` // ForeignKey should reference the field in the LGA table
	IncidentReportRating string    `json:"incident_report" gorm:"foreignKey:IncidentReportRating"` // ForeignKey should reference the field in the Rating table
	DateOfIncidence      time.Time `json:"date_of_incidence" gorm:"column:date_of_incidence"` // No foreignKey tag needed for DateOfIncidence
}

type SubReport struct {
    ID                  string `json:"id" gorm:"primaryKey;index"`
    ReportTypeID        string `json:"report_type_id" gorm:"foreignKey:ID"`
    LGAID               string `json:"lga_id" gorm:"foreignKey:ID"`
    UserID              uint   `json:"user_id" gorm:"foreignKey:UserID"`
    StateName           string `json:"state_name" gorm:"foreignKey:StateName"`
    SubReportName       string `json:"sub_report_name"`
    ReportTypeCategory  string `json:"report_type_category" gorm:"foreignKey:Category"`
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
