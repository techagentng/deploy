package models

// Reward represents rewards earned by users
type Post struct {
	Model
	UserID          uint   `json:"user_id" gorm:"foreignKey:ID"`
	Title           string `json:"post"`
	PostCategory    string `json:"post_category"`
	Image           string `json:"post_image"`
	PostDescription string `json:"post_description"`
	UserFullname         string     `json:"fullname"`
}
