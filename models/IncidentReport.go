package models

import (
	"time"

	"github.com/google/uuid"
)

type IncidentReport struct {
	ID                   uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"` // Update to UUID type
	CreatedAt            int64      `json:"created_at"`
	UserFullname         string     `json:"fullname"`
	DateOfIncidence      string     `json:"date_of_incidence"`
	Description          string     `json:"description" gorm:"type:varchar(1000)"`
	FeedURLs             string     `json:"feed_urls"`
	VideoURL             string     `json:"video_url"`
	AudioURL             string     `json:"audio_url"`
	ThumbnailURLs        string     `json:"thumbnail_urls"`
	FullSizeURLs         string     `json:"full_size_urls"`
	ProductName          string     `json:"product_name"`
	StateName            string     `json:"state_name"`
	LGAName              string     `json:"lga_name"`
	Latitude             float64    `json:"latitude"`
	Longitude            float64    `json:"longitude"`
	UserIsAnonymous      bool       `json:"user_is_anonymous"`
	Address              string     `json:"address"`
	UserUsername         string     `json:"username"`
	Telephone            string     `json:"telephone"`
	Email                string     `json:"email"`
	View                 int        `json:"view"`
	IsVerified           bool       `json:"is_verified"`
	UserID               uint       `json:"user_id"`
	AdminID              uint       `json:"is_admin"`
	Landmark             string     `json:"landmark"`
	LikeCount            int        `json:"like_count"`
	BookmarkedReports    []*User    `gorm:"many2many:incident_report_user;" json:"bookmarked_reports"`
	IsResponse           bool       `json:"is_response"`
	TimeofIncidence      time.Time  `json:"time_of_incidence"`
	ReportStatus         string     `json:"report_status"`
	BlockRequest         string     `json:"block_request"`
	RewardPoint          int        `json:"reward_point"`
	RewardAccountNumber  string     `json:"reward_account_number"`
	ActionTypeName       string     `json:"action_type_name"`
	IsState              bool       `json:"is_state"`
	Rating               string     `json:"rating"`
	HospitalName         string     `json:"hospital_name"`
	Department           string     `json:"department"`
	DepartmentHeadName   string     `json:"department_head_name"`
	AccidentCause        string     `json:"accident_cause"`
	SchoolName           string     `json:"school_name"`
	VicePrincipal        string     `json:"vice_principal"`
	OutageLength         string     `json:"outage_length"`
	AirportName          string     `json:"airport_name"`
	Country              string     `json:"country"`
	StateEmbassyLocation string     `json:"state_embassy_location"`
	NoWater              bool       `json:"no_water"`
	AmbassedorsName      string     `json:"ambassedors_name"`
	HospitalAddress      string     `json:"hospital_address"`
	RoadName             string     `json:"road_name"`
	AirlineName          string     `json:"airline_name"`
	Category             string     `json:"category"`
	Terminal             string     `json:"terminal"`
	QueueTime            string     `json:"queue_time"`
	SubReportType        string     `json:"sub_report_type"`
	UpvoteCount          int        `json:"upvote_count" gorm:"default:0"`
	DownvoteCount        int        `json:"downvote_count" gorm:"default:0"`
	ReportTypeID         uuid.UUID  `json:"report_type_id"`
	ReportType           ReportType `gorm:"foreignKey:ReportTypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Followers []*User `gorm:"many2many:follows;joinForeignKey:ReportID;joinReferences:UserID" json:"followers"`
	IsAnonymous bool `json:"is_anonymous" gorm:"column:is_anonymous"`
}

type ReportCount struct {
	StateName string
	LGAName   string
	Count     int
}

// IncidentReportUser defines the relationship between users and incident reports (i.e., bookmarks)

type IncidentReportUser struct {
	ID               uint      `gorm:"primaryKey;autoIncrement"`
	UserID           uint      `gorm:"not null"`
	IncidentReportID string    `gorm:"not null;type:varchar(36);index"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
	User             User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
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

type ReportPostRequest struct {
	ReportTypeID int    `json:"report_type_id"`
	SubReportID  int    `json:"sub_report_id"`
	UserID       int    `json:"user_id"`
	Description  string `json:"description"`
	Message      string `json:"message"`
}
