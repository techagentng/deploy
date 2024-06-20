package models

// Reward represents rewards earned by users
type Reward struct {
	Model
	IncidentReportID string `json:"incident_id" gorm:"foreignKey:ID"`
	UserID           uint   `json:"user_id" gorm:"foreignKey:UserID"`
	RewardType       string `json:"reward_type"`
	Point            int    `json:"point"`
	Balance          int    `json:"balance"`
	AccountNumber    string `json:"account_number"`
}
