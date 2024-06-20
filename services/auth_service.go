package services

import (
	"crypto/rand"
	// "encoding/json"
	"errors"
	"fmt"

	// "io/ioutil"
	_ "github.com/gin-gonic/gin"
	_ "github.com/golang-jwt/jwt"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	apiError "github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/services/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"net/http"
)

//go:generate mockgen -destination=../mocks/auth_mock.go -package=mocks github.com/decagonhq/meddle-api/services AuthService

// AuthService interface
type AuthService interface {
	LoginUser(request *models.LoginRequest) (*models.LoginResponse, *apiError.Error)
	LoginMacAddressUser(loginRequest *models.LoginRequestMacAddress) (*models.LoginRequestMacAddress, *apiError.Error)
	SignupUser(request *models.User) (*models.User, *apiError.Error)
	UpdateUserImageUrl(request *models.User, filepart string) *apiError.Error
	GetUserProfile(userID uint) (*models.User, error)
	EditUserProfile(userID uint, userDetails *models.EditProfileResponse) error
	// FacebookSignInUser(token string) (*string, *apiError.Error)
	// VerifyEmail(token string) error
	SendEmailForPasswordReset(user *models.ForgotPassword) *apiError.Error
	ResetPassword(user *models.ResetPassword, token string) *apiError.Error
	GetAllUsers() ([]models.User, error)
	// DeleteUserByEmail(userEmail string) *apiError.Error
}

// authService struct
type authService struct {
	Config   *config.Config
	authRepo db.AuthRepository
}

// NewAuthService instantiate an authService
func NewAuthService(authRepo db.AuthRepository, conf *config.Config) AuthService {
	return &authService{
		Config:   conf,
		authRepo: authRepo,
	}
}

func (a *authService) SignupUser(user *models.User) (*models.User, *apiError.Error) {
	// userID := uuid.New().String()
	// user.ID = userID
	err := a.authRepo.IsEmailExist(user.Email)
	if err != nil {
		// FIXME: return the proper error message from the function
		// TODO: handle internal server error later
		return nil, apiError.New("email already exist", http.StatusBadRequest)
	}

	err = a.authRepo.IsPhoneExist(user.Telephone)
	if err != nil {
		fmt.Println("called 404")
		return nil, apiError.New("phone already exist", http.StatusBadRequest)
	}

	user.HashedPassword, err = GenerateHashPassword(user.Password)
	if err != nil {
		log.Printf("error generating password hash: %v", err.Error())
		return nil, apiError.New("internal server error", http.StatusInternalServerError)
	}

	// token, err := jwt.GenerateToken(user.Email, a.Config.JWTSecret)
	// if err != nil {
	// 	return nil, apiError.New("internal server error", http.StatusInternalServerError)
	// }
	// if err := a.sendVerifyEmail(token, user.Email); err != nil {
	// 	return nil, err
	// }

	user.Password = ""
	user.IsEmailActive = true
	user, err = a.authRepo.CreateUser(user)

	if err != nil {
		log.Printf("unable to create user: %v", err.Error())
		return nil, apiError.New("internal server error", http.StatusInternalServerError)
	}

	return user, nil
}

func GenerateHashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func (a *authService) LoginMacAddressUser(loginRequest *models.LoginRequestMacAddress) (*models.LoginRequestMacAddress, *apiError.Error) {
	// Generate MAC address token
	macAddressToken, err := jwt.GenerateMacAddressToken(loginRequest.MacAddress, a.Config.JWTSecret)
	if err != nil {
		log.Printf("error generating MAC address token: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	// Return the MAC address token in the login response
	return &models.LoginRequestMacAddress{
		MacAddress: macAddressToken,
	}, nil
}

// LoginUser logs in a user and returns the login response
func (a *authService) LoginUser(loginRequest *models.LoginRequest) (*models.LoginResponse, *apiError.Error) {
	foundUser, err := a.authRepo.FindUserByEmail(loginRequest.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apiError.New("invalid email or password", http.StatusUnprocessableEntity)
		} else {
			log.Printf("error from database: %v", err)

			return nil, apiError.New("unable to find user", http.StatusInternalServerError)
		}
	}

	// if !foundUser.IsEmailActive {
	// 	return nil, apiError.New("email not verified", http.StatusUnauthorized)
	// }

	if err := foundUser.VerifyPassword(loginRequest.Password); err != nil {
		return nil, apiError.ErrInvalidPassword
	}

	accessToken, refreshToken, err := jwt.GenerateTokenPair(foundUser.Email, a.Config.JWTSecret, foundUser.AdminStatus, foundUser.ID)
	if err != nil {
		log.Printf("error generating token pair: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	return &models.LoginResponse{
		UserResponse: models.UserResponse{
			ID:        foundUser.ID,
			Fullname:  foundUser.Fullname,
			Username:  foundUser.Username,
			Telephone: foundUser.Telephone,
			Email:     foundUser.Email,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *authService) VerifyEmail(token string) error {
	claims, err := jwt.ValidateAndGetClaims(token, a.Config.JWTSecret)
	if err != nil {
		return apiError.New("invalid link", http.StatusUnauthorized)
	}
	email := claims["email"].(string)
	err = a.authRepo.VerifyEmail(email, token)
	return err
}

// func (a *authService) GetUserByID(id string) (*models.User, error) {
//     user, err := a.authRepo.FindByID(id)
//     if err != nil {
//         return nil, err
//     }
//     return user, nil
// }

func (a *authService) UpdateUserImageUrl(user *models.User, imagePath string) *apiError.Error {
	// Update user's profile with the image URL
	user.ThumbNailURL = imagePath
	err := a.authRepo.UpdateUserImage(user)
	if err != nil {
		return &apiError.Error{
			Message: "Failed to update user profile",
			Status:  0,
		}
	}
	return nil
}

func GenerateRandomString() (string, error) {
	n := 5
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := fmt.Sprintf("%X", b)
	return s, nil
}

func (a *authService) GetUserProfile(userID uint) (*models.User, error) {
	// Call repository method to fetch user profile
	user, err := a.authRepo.FindUserByID(userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a *authService) EditUserProfile(userID uint, userDetail *models.EditProfileResponse) error {
	// Implement your business logic here, if needed
	// For example, you might want to perform validation on the user details before updating

	// Call the repository method to update user profile
	return a.authRepo.EditUserProfile(userID, userDetail)
}

func (a *authService) SendEmailForPasswordReset(user *models.ForgotPassword) *apiError.Error {
	return apiError.ErrBadRequest
}

func (a *authService) ResetPassword(user *models.ResetPassword, token string) *apiError.Error {
	return apiError.ErrBadRequest
}

func (s *authService) GetAllUsers() ([]models.User, error) {
	users, err := s.authRepo.GetAllUsers()
	if err != nil {
		return nil, fmt.Errorf("error getting all users: %w", err)
	}
	return users, nil
}
