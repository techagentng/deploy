package models

type Blacklist struct {
	Model
	Token string `json:"token"`
	Email string `json:"email"`
}
