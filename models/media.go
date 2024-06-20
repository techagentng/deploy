package models

type Media struct {
	ID string `json:"id"`
	// IncidentReportID string `json:"incident_id" gorm:"column:incident_report_id"`
	FileType     string `json:"file_type"`
	FileSize     int64  `json:"file_size"`
	Filename     string `json:"file_name"`
	UserID       uint   `gorm:"foreignKey:ID"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FeedURL      string `json:"feed_url"`
	FullSizeURL  string `json:"full_size_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Count        int    `json:"count"`
	Points       int    `json:"points"`
}

type MediaCount struct {
	Model
	Images           int
	Videos           int
	Audios           int
	UserID           uint
	IncidentReportID string
}
