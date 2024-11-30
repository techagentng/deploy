// I removed the verify password by encryption
package models

import (
	"errors"
	"fmt"
	"time"

	goval "github.com/go-passwd/validator"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/leebenson/conform"
)

// User represents a user of the application
type User struct {
	Model
	Fullname          string            `json:"fullname" binding:"required,min=2"`
	Username          string            `json:"username" binding:"required,min=2"`
	Telephone string `json:"telephone" gorm:"default:null" binding:"required"`
	Email             string            `json:"email" gorm:"unique;not null" binding:"required,email"`
	IsQueried         bool              `json:"is_queried" gorm:"default:false"`
	IsBlocked         bool              `json:"is_blocked" gorm:"default:false"`
	Password string `json:"password,omitempty" validate:"omitempty,min=4"`
	HashedPassword    string            `json:"-"`
	IsEmailActive     bool              `json:"-"`
	IsSocial          bool              `json:"-"`
	AccessToken       string            `json:"-"`
	IsVerified        bool              `json:"is_verified"`
	IsAnonymous       bool              `json:"is_anonymous"`
	IsJournalist      bool              `json:"is_journalist"`
	AdminStatus       bool              `json:"is_admin" gorm:"foreignKey:Status"` // admin
	Notifications     []Notification    `gorm:"foreignKey:UserID"`
	ThumbNailURL      string            `json:"thumbnail_url,omitempty"`
	MacAddress        string            `json:"mac_address"`
	ResetToken	      string             `json:"reset_token"`
	LGAName           string            `gorm:"foreignKey:Name"`
	Online            bool              `json:"online"`
	Upvotes           int               `json:"up_vote"`
	Downvotes         int               `json:"down_vote"`
	RoleID            uuid.UUID         `gorm:"type:uuid" json:"role_id"`
	Role              Role              `gorm:"foreignKey:RoleID" json:"role"`
	BookmarkedReports []*IncidentReport `gorm:"many2many:incident_report_user;" json:"bookmarked_reports"`
}

type ReportUserRequest struct {
	UserID  uuid.UUID `json:"user_id" binding:"required"`
	Message string    `json:"message" binding:"required"`
}

type Admin struct {
	Model
	Status bool `json:"is_admin"`
}

type UserPoints struct {
	Model
	UserID string `json:"user_id"`
	Points int    `json:"points"`
}

// CreateSocialUserParams represents the parameters required to create a new social user.
type CreateSocialUserParams struct {
	Email    string `json:"email"`
	IsSocial bool   `json:"is_social"`
	Active   bool   `json:"active"`
	Name     string `json:"name"`
}

//	func ValidateStruct(req interface{}) []error {
//		validate := validator.New()
//		// english := en.New()
//		// uni := ut.New(english, english)
//		// trans, _ := uni.GetTranslator("en")
//		// _ = enTranslations.RegisterTranslationsFunc(validattrans)
//		err := validateWhiteSpaces(req)
//		errs := translateError(err, trans)
//		err = validate.Struct(req)
//		errs = translateError(err, trans)
//		return errs
//	}
func ValidatePassword(password string) error {
	passwordValidator := goval.New(goval.MinLength(6, errors.New("password cant be less than 6 characters")),
		goval.MaxLength(15, errors.New("password cant be more than 15 characters")))
	err := passwordValidator.Validate(password)
	return err
}
func validateWhiteSpaces(data interface{}) error {
	return conform.Strings(data)
}

func translateError(err error, trans ut.Translator) (errs []error) {
	if err == nil {
		return nil
	}
	validatorErrs := err.(validator.ValidationErrors)
	for _, e := range validatorErrs {
		translatedErr := fmt.Errorf(e.Translate(trans) + "; ")
		errs = append(errs, translatedErr)
	}
	return errs
}

type UserResponse struct {
	ID        uint   `json:"id"`
	Fullname  string `json:"fullname"`
	Username  string `json:"username"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
	LGA       string `json:"LGA" gorm:"foreignkey:LGA(id)"`
	RoleName  string `json:"role_name"`
}

type UserImage struct {
	ID           uint `gorm:"primaryKey"`
	UserID       uint
	ThumbNailURL string
	CreatedAt    time.Time
}

type EditProfileResponse struct {
	ID       uint   `json:"id"`
	FullName string `json:"fullname"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	State    string `json:"state"`
	Lga      string `json:"lga"`
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginRequestMacAddress struct {
	Model
	MacAddress string `json:"mac_address"`
	Token      string `json:"token"`
}
type ForgotPassword struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}
type UserIdResponse struct {
	ID uint `json:"id"`
}

type ResetPassword struct {
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}
type GoogleAuthResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type LoginResponse struct {
	UserResponse
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	RoleID       string `json:"role_id"`
}

// VerifyPassword verifies the collected password with the user's hashed password
//
//	func (u *User) VerifyPassword(password string) error {
//		return bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(password))
//	}
func (u *User) VerifyPassword(password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(password))
	if err != nil {
		return err // Passwords do not match
	}
	return nil // Passwords match
}
