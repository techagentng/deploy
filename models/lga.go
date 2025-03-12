package models

import "github.com/google/uuid"

// LGA struct
type LGA struct {
    ID      uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
    Name    *string   `gorm:"default:null" json:"name"`
    StateID uuid.UUID `gorm:"type:uuid;not null" json:"state_id"`
    State   State     `gorm:"foreignKey:StateID" json:"state"`
}

type State struct {
    ID            uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
    State         *string   `gorm:"default:null" json:"state"`
    Governor      *string   `gorm:"default:null" json:"governor"`
    DeputyName    *string   `gorm:"default:null" json:"deputy_name"`
    DeputyImage   *string   `gorm:"default:null" json:"deputy_image"`
    LGAC          *string   `gorm:"default:null" json:"lgac"`
    LgacImage     *string   `gorm:"default:null" json:"lgac_image"`
    GovernorImage *string   `gorm:"default:null" json:"governor_image"`
}


type LGAReportCount struct {
	LGAName     string `gorm:"default:null"json:"lga_name"`
	ReportCount int    `gorm:"default:null"json:"report_count"`
}
