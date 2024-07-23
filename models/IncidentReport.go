package models

import "time"

// IncidentReport represents an incident report filed by a user
type IncidentReport struct {
	ID                   string    `json:"id" gorm:"primaryKey;index"`
	CreatedAt            int64     `json:"created_at"`
	UserFullname         string    `json:"fullname" gorm:"foreignKey:Fullname"`
	DateOfIncidence      string `json:"date_of_incidence"`
	Description          string    `json:"description" gorm:"type:varchar(1000)"`
	FeedURLs             string    `json:"feed_urls"`
	ThumbnailURLs        string    `json:"thumbnail_urls"`
	FullSizeURLs         string    `json:"full_size_urls"`
	ProductName          string    `json:"product_name"`
	StateName            string    `json:"state_name" gorm:"foreignKey:Name"`
	LGAName              string    `json:"lga_name" gorm:"foreignKey:Name"`
	Latitude             float64   `json:"latitude"`
	Longitude            float64   `json:"longitude"`
	UserIsAnonymous      bool      `json:"user_is_anonymous" gorm:"foreignKey:IsAnonymous"`
	Address              string    `json:"address"`
	UserUsername         string    `json:"username" gorm:"foreignKey:Username" binding:"required"`
	Telephone            string    `json:"telephone"`
	Email                string    `json:"email"`
	View                 int       `json:"view"`
	IsVerified           bool      `json:"is_verified"`
	UserID               uint      `json:"user_id" gorm:"foreignKey:ID"`
	ReportTypeID         string    `json:"report_type_id" gorm:"foreignKey:ID"`
	AdminID              uint      `json:"is_admin" gorm:"foreignKey:Status"` // admin
	Landmark             string    `json:"landmark"`
	LikeCount            int       `json:"like_count" gorm:"foreignKey:Count"`
	BookmarkedReports    []*User   `gorm:"many2many:incident_report_user;" json:"bookmarked_reports"`
	IsResponse           bool      `json:"is_response"`
	TimeofIncidence      time.Time `json:"time_of_incidence"`
	ReportStatus         string    `json:"report_status"`
	RewardPoint          int       `json:"reward_point" gorm:"foreignKey:Point"`
	RewardAccountNumber  string    `json:"reward_account_number" gorm:"foreignKey:AccountNumber"`
	ActionTypeName       string    `json:"action_type_name" gorm:"foreignKey:Name"`
	ReportTypeName       string    `json:"report_type_name" gorm:"foreignKey:Name"`
	IsState              bool      `json:"is_state"`
	Rating               string    `json:"rating"`
	HospitalName         string    `json:"hospital_name"`
	Department           string    `json:"department"`
	DepartmentHeadName   string    `json:"department_head_name"`
	AccidentCause        string    `json:"accident_cause"`
	SchoolName           string    `json:"school_name"`
	VicePrincipal        string    `json:"vice_principal"`
	OutageLength         string    `json:"outage_length"`
	AirportName          string    `json:"airport_name"`
	Country              string    `json:"country"`
	StateEmbassyLocation string    `json:"state_embassy_location"`
	NoWater              bool      `json:"no_water"`
	AmbassedorsName      string    `json:"ambassedors_name"`
	HospitalAddress      string    `json:"hospital_address"`
	RoadName             string    `json:"road_name"`
	AirlineName          string    `json:"airline_name"`
	Category            string    `json:"category"`
	Terminal            string    `json:"terminal"`
	QueueTime           string    `json:"queue_time"`
}

type ReportCount struct {
    StateName string
    LGAName   string
    Count     int
}

type Actions struct {
	Model
	ActionType string `json:"action_type"`
}

type StateReportPercentage struct {
	State      string  `json:"state"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type BookmarkedReport struct {
	UserID           uint `json:"user_id" gorm:"foreignKey:ID"`         // Foreign key referencing User model
	IncidentReportID uint `gorm:"column:incident_report_id;primaryKey"` // Foreign key referencing IncidentReport model
}

// IncidentReportResponseData represents the response data for saving incident reports
type IncidentReportResponseData struct {
	ID              uint   `json:"id"`
	ReportType      string `json:"report_type"`
	State           string `json:"state"`
	DateofIncidence string `json:"date_of_incidence"`
	TimeofIncidence string `json:"time_of_incidence"`
	Landmark        string `json:"landmark"`
	ImageURL        string `json:"image_url"`
}
