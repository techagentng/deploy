package models

// Reward represents rewards earned by users
type Votes struct {
	Model
	UserID   uint   `json:"user_id" gorm:"foreignKey:UserID"`
	ReportID string `json:"report_type_id"`
	VoteType string `json:"vote_type"`
}
