package models

// Comment represents a user's comment on an incident report
type Comment struct {
	Model
	Content          string `json:"comment"`
	IncidentReportID uint
	UserID           uint
}
