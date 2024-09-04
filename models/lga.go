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
    ID   uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
    Name string    `gorm:"not null"`
}



type LGAReportCount struct {
    LGAName     string `json:"lga_name"`
    ReportCount int    `json:"report_count"`
}