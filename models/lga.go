package models

import "github.com/google/uuid"

// LGA struct
type LGA struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	 Name *string `gorm:"default:null"`
	StateID uuid.UUID `gorm:"type:uuid;not null"`
	State   State     `gorm:"foreignKey:StateID" json:"state"`
}

type State struct {
    ID       uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
    State     string    `gorm:"" json:"state"`
    Governor string    `gorm:"" json:"governor"`
	DeputyName string    `gorm:"" json:"deputy_name"`
	DeputyImage string `gorm:"" json:"deputy_image"`
    LGAC   string    `gorm:"" json:"lgac"`
	LgacImage   string `gorm:"" json:"lgac_image"` 
    GovernorImage string `gorm:"" json:"governor_image"`
	       
}


type LGAReportCount struct {
	LGAName     string `json:"lga_name"`
	ReportCount int    `json:"report_count"`
}
