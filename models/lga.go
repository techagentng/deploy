package models

import "github.com/google/uuid"

// LGA struct
type LGA struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name    string    `gorm:"not null"`
	StateID uuid.UUID `gorm:"type:uuid;not null"`
	State   State     `gorm:"foreignKey:StateID" json:"state"`
}

type State struct {
    ID       uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
    StateName     string    `gorm:"" json:"state_name"`
    Governor string    `gorm:"" json:"governor"`
    LGAC   string    `gorm:"" json:"lgac"`
    GovernorImage string `gorm:"" json:"governor_image"`  // Image URL for the Governor
    LgacImage   string `gorm:"" json:"lgac_image"`    // Image URL for the Deputy
}


type LGAReportCount struct {
	LGAName     string `json:"lga_name"`
	ReportCount int    `json:"report_count"`
}
