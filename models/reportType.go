package models

import (
	"time"

	"github.com/google/uuid"
)

type ReportType struct {
	ID                   uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	UserID               uint             `json:"user_id"`
	IncidentReportID     uuid.UUID        `json:"incident_report_id"`
	IncidentReports      []IncidentReport `gorm:"foreignKey:ReportTypeID;references:ID"`
	Category             string           `json:"category" binding:"required"`
	Name                 string           `json:"name" binding:"required"` // Added Name field
	StateName            string           `json:"state_name"`
	LGAName              string           `json:"lga_name"`
	IncidentReportRating string           `json:"incident_report_rating"`
	DateOfIncidence      time.Time        `json:"date_of_incidence"`
	SubReports           []SubReport      `gorm:"foreignKey:ReportTypeID"`
	CreatedAt            time.Time        `gorm:"autoCreateTime" json:"created_at"`
}


type SubReport struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	ReportTypeID     uuid.UUID  `gorm:"type:uuid;not null" json:"report_type_id"`
	SubReportType    string     `json:"sub_report_type"`
	Description      string     `json:"description"`
	ReportType       ReportType `gorm:"foreignKey:ReportTypeID"`
	IncidentReportID uuid.UUID  `json:"incident_report_id"`
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
