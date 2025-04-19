package services

import (
	"crypto/rand"
	"strings"
	// "encoding/json"
	"errors"
	"fmt"

	// "io/ioutil"
	"log"
	"net/http"

	_ "github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	apiError "github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/services/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

//go:generate mockgen -destination=../mocks/auth_mock.go -package=mocks github.com/decagonhq/meddle-api/services AuthService

// AuthService interface
type AuthService interface {
	LoginUser(loginRequest *models.LoginRequest) (*models.LoginResponse, *apiError.Error)
	LoginMacAddressUser(loginRequest *models.LoginRequestMacAddress) (*models.LoginRequestMacAddress, *apiError.Error)
	SignupUser(request *models.User) (*models.User, error)
	// UpdateUserImageUrl(imagePath string) *apiError.Error
	GetUserProfile(userID uint) (*models.User, error)
	EditUserProfile(userID uint, userDetails *models.EditProfileResponse) error
	// FacebookSignInUser(token string) (*string, *apiError.Error)
	// VerifyEmail(token string) error
	SendEmailForPasswordReset(user *models.ForgotPassword) *apiError.Error
	ResetPassword(user *models.ResetPassword, token string) *apiError.Error
	GetAllUsers() ([]models.User, error)
	// DeleteUserByEmail(userEmail string) *apiError.Error
	GetRoleByName(name string) (*models.Role, error)
	DeleteUser(userID uint) error
	GoogleLoginUser(loginRequest *models.GoogleLoginRequest) (*models.LoginResponse, *apiError.Error)
	FacebookLoginUser(loginRequest *models.FacebookLoginRequest) (*models.LoginResponse, *apiError.Error)
	
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

func (s *authService) SignupUser(user *models.User) (*models.User, error) {
	if user == nil {
		log.Println("SignupUser error: user is nil")
		return nil, errors.New("user is nil")
	}

	if user.Email == "" {
		log.Println("SignupUser error: email is empty")
		return nil, errors.New("email is empty")
	}

	// Check if the email already exists
	err := s.authRepo.IsEmailExist(user.Email)
	if err != nil {
		log.Printf("SignupUser error: %v", err)
		return nil, apiError.GetUniqueContraintError(err)
	}

	// Check if the phone number already exists
	err = s.authRepo.IsPhoneExist(user.Telephone)
	if err != nil {
		log.Printf("SignupUser error: %v", err)
		return nil, apiError.GetUniqueContraintError(err)
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("SignupUser error hashing password: %v", err)
		return nil, apiError.ErrInternalServerError
	}
	user.HashedPassword = string(hashedPassword)
	user.Password = "" // Clear the plain password

	// Create the user in the database
	user, err = s.authRepo.CreateUser(user)
	if err != nil {
		log.Printf("SignupUser error creating user: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	// Fetch the created user
	createdUser, err := s.authRepo.FindUserByEmail(user.Email)
	if err != nil {
		log.Printf("SignupUser error fetching created user: %v", err)
		return nil, apiError.ErrInternalServerError
	}

	return createdUser, nil
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
		MacAddress: loginRequest.MacAddress, // Original MAC address
		Token:      macAddressToken,         // Include the token
	}, nil
}

// LoginUser logs in a user and returns the login response
func (a *authService) LoginUser(loginRequest *models.LoginRequest) (*models.LoginResponse, *apiError.Error) {
    // Find the user by email
    foundUser, err := a.authRepo.FindGoogleUserByEmail(loginRequest.Email)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, apiError.New("invalid email or password", http.StatusUnprocessableEntity)
        }
        log.Printf("Error finding user by email: %v", err)
        return nil, apiError.New("unable to find user", http.StatusInternalServerError)
    }

    // Verify user password
    if err := foundUser.VerifyPassword(loginRequest.Password); err != nil {
        log.Printf("Invalid password for user %s", foundUser.Email)
        return nil, apiError.ErrInvalidPassword
    }

    // Ensure RoleID is not empty
    if foundUser.RoleID == uuid.Nil {
        log.Printf("User %s does not have a role assigned", foundUser.Email)
        return nil, apiError.New("user role not assigned", http.StatusInternalServerError)
    }

    // Fetch the user's role
    log.Printf("Fetching role with ID: %s for user %s", foundUser.RoleID.String(), foundUser.Email)
    role, err := a.authRepo.FindRoleByID(foundUser.RoleID)
    if err != nil {
        log.Printf("Error fetching role for user %s: %v", foundUser.Email, err)
        return nil, apiError.New("unable to fetch role", http.StatusInternalServerError)
    }

    roleName := role.Name

    // ========== PUSH TOKEN STORAGE ========== //
    if loginRequest.ExpoPushToken != "" {
        // Update user's push token in database
        if err := a.authRepo.UpdateUserExpoPushToken(foundUser.ID, loginRequest.ExpoPushToken); err != nil {
            log.Printf("Failed to update push token for user %s: %v", foundUser.Email, err)
            // Don't fail login - just log the error
        } else {
            log.Printf("Updated Expo push token for user %s", foundUser.Email)
        }
    }

    // Generate tokens with role information
    log.Printf("Generating token pair for user %s with role %s", foundUser.Email, roleName)
    accessToken, refreshToken, err := jwt.GenerateTokenPair(foundUser.Email, a.Config.JWTSecret, foundUser.AdminStatus, foundUser.ID, roleName)
    if err != nil {
        log.Printf("Error generating token pair for user %s: %v", foundUser.Email, err)
        return nil, apiError.ErrInternalServerError
    }

    return &models.LoginResponse{
        UserResponse: models.UserResponse{
            ID:        foundUser.ID,
            Fullname:  foundUser.Fullname,
            Username:  foundUser.Username,
            Telephone: foundUser.Telephone,
            Email:     foundUser.Email,
            RoleName:  roleName,
        },
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}

// DefaultUserRoleID is the predefined UUID for the "user" role
var DefaultUserRoleID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

func (a *authService) GoogleLoginUser(loginRequest *models.GoogleLoginRequest) (*models.LoginResponse, *apiError.Error) {
    foundUser, err := a.authRepo.FindGoogleUserByEmail(loginRequest.Email)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return a.createGoogleUser(loginRequest.Email)
        }
        log.Printf("Error finding user by email %s: %v", loginRequest.Email, err)
        return nil, apiError.New("unable to find user", http.StatusInternalServerError)
    }

    roleName := "user" // Default for social logins
    if foundUser.RoleID != uuid.Nil {
        role, err := a.authRepo.FindRoleByID(foundUser.RoleID)
        if err != nil {
            log.Printf("Error fetching role for user %s: %v, defaulting to 'user'", foundUser.Email, err)
            roleName = "user" // Fallback to default
        } else {
            roleName = role.Name
        }
    }

    accessToken, refreshToken, err := jwt.GenerateTokenPair(foundUser.Email, a.Config.JWTSecret, foundUser.AdminStatus, foundUser.ID, roleName)
    if err != nil {
        log.Printf("Error generating token pair for user %s: %v", foundUser.Email, err)
        return nil, apiError.ErrInternalServerError
    }

    return &models.LoginResponse{
        UserResponse: models.UserResponse{
            ID:        foundUser.ID,
            Fullname:  foundUser.Fullname,
            Username:  foundUser.Username,
            Telephone: foundUser.Telephone,
            Email:     foundUser.Email,
            RoleName:  roleName,
        },
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}

func (a *authService) createGoogleUser(email string) (*models.LoginResponse, *apiError.Error) {
    username := strings.Split(email, "@")[0]
    if len(username) < 2 {
        username = username + "user"
    }

    // Ensure username uniqueness
    baseUsername := username
    for i := 1; ; i++ {
        existingUser, err := a.authRepo.FindGoogleUserByUsername(username)
        if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
            log.Printf("Error checking username %s: %v", username, err)
            return nil, apiError.New("unable to verify username", http.StatusInternalServerError)
        }
        if existingUser == nil {
            break
        }
        username = fmt.Sprintf("%s%d", baseUsername, i)
    }

    // Use the predefined "user" role ID directly
    newUser := &models.User{
        Email:     email,
        Username:  username,
        Telephone: "", // Matches your struct's gorm:"default:null"
        IsSocial:  true,
        RoleID:    DefaultUserRoleID,
    }

    if err := a.authRepo.GoogleUserCreate(newUser); err != nil {
        log.Printf("Error creating user for email %s: %v", email, err)
        return nil, apiError.New(fmt.Sprintf("unable to create user: %v", err), http.StatusInternalServerError)
    }

    roleName := "user"
    accessToken, refreshToken, err := jwt.GenerateTokenPair(newUser.Email, a.Config.JWTSecret, newUser.AdminStatus, newUser.ID, roleName)
    if err != nil {
        log.Printf("Error generating token pair for user %s: %v", email, err)
        return nil, apiError.ErrInternalServerError
    }

    return &models.LoginResponse{
        UserResponse: models.UserResponse{
            ID:        newUser.ID,
            Fullname:  newUser.Fullname,
            Username:  newUser.Username,
            Telephone: newUser.Telephone,
            Email:     newUser.Email,
            RoleName:  roleName,
        },
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}
// Example helper to get a default role ID
func (a *authService) getDefaultRoleID() (uuid.UUID, error) {
    role, err := a.authRepo.FindRoleByName("user") // Assuming a "user" role exists
    if err != nil {
        return uuid.Nil, err
    }
    return role.ID, nil
}
// func (a *authService) GetUserByID(id string) (*models.User, error) {
//     user, err := a.authRepo.FindByID(id)
//     if err != nil {
//         return nil, err
//     }
//     return user, nil
// }

// func (a *authService) UpdateUserImageUrl(imagePath string) *apiError.Error {
// 	// Update user's profile with the image URL
// 	var user models.User
// 	user.ThumbNailURL = imagePath

// 	err := a.authRepo.UpdateUserImage(&user)
// 	if err != nil {
// 		log.Printf("Error updating user image in database: %v", err)
// 		return &apiError.Error{
// 			Message: "Failed to update user profilxxe",
// 			Status:  0,
// 		}
// 	}
// 	return nil
// }

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

// GetRoleByName retrieves a role from the repository by its name.
func (a *authService) GetRoleByName(name string) (*models.Role, error) {
	// Call the repository method to fetch the role
	role, err := a.authRepo.FindRoleByName(name)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (s *authService) DeleteUser(userID uint) error {
	return s.authRepo.SoftDeleteUser(userID)
}

func (a *authService) FacebookLoginUser(loginRequest *models.FacebookLoginRequest) (*models.LoginResponse, *apiError.Error) {
    // Find the user by email
    foundUser, err := a.authRepo.FindFacebookUserByEmail(loginRequest.Email)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            // Create a new user if they don’t exist
            return a.createFacebookUser(loginRequest.Email, loginRequest.Fullname, loginRequest.Telephone)
        }
        log.Printf("Error finding user by email: %v", err)
        return nil, apiError.New("unable to find user", http.StatusInternalServerError)
    }

    // Fetch role name, defaulting to "user"
    roleName := "user"
    if foundUser.RoleID != uuid.Nil {
        role, err := a.authRepo.FindRoleByID(foundUser.RoleID)
        if err != nil {
            log.Printf("Error fetching role for user %s: %v", foundUser.Email, err)
            return nil, apiError.New("unable to fetch role", http.StatusInternalServerError)
        }
        roleName = role.Name
    }

    // Generate tokens with role information
    log.Printf("Generating token pair for user %s with role %s", foundUser.Email, roleName)
    accessToken, refreshToken, err := jwt.GenerateTokenPair(foundUser.Email, a.Config.JWTSecret, foundUser.AdminStatus, foundUser.ID, roleName)
    if err != nil {
        log.Printf("Error generating token pair for user %s: %v", foundUser.Email, err)
        return nil, apiError.ErrInternalServerError
    }

    return &models.LoginResponse{
        UserResponse: models.UserResponse{
            ID:        foundUser.ID,
            Fullname:  foundUser.Fullname,
            Username:  foundUser.Username,
            Telephone: foundUser.Telephone,
            Email:     foundUser.Email,
            RoleName:  roleName,
        },
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}

// createFacebookUser creates a new Facebook user
func (a *authService) createFacebookUser(email, fullname, telephone string) (*models.LoginResponse, *apiError.Error) {
    username := strings.Split(email, "@")[0]
    if len(username) < 2 {
        username = username + "user"
    }

    // Ensure username uniqueness
    baseUsername := username
    for i := 1; ; i++ {
        existingUser, err := a.authRepo.FindFacebookUserByUsername(username)
        if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
            log.Printf("Error checking username %s: %v", username, err)
            return nil, apiError.New("unable to verify username", http.StatusInternalServerError)
        }
        if existingUser == nil {
            break
        }
        username = fmt.Sprintf("%s%d", baseUsername, i)
    }

    // Fetch or create the "user" role
    role, err := a.authRepo.FindRoleByName("user")
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            role = &models.Role{
                ID:   uuid.New(),
                Name: "user",
            }
            role, err = a.authRepo.CreateRole(role)
            if err != nil {
                return nil, apiError.New(fmt.Sprintf("unable to create role: %v", err), http.StatusInternalServerError) // Propagate CreateRole error
            }
        } else {
            log.Printf("Error fetching 'user' role: %v", err)
            return nil, apiError.New("unable to assign role", http.StatusInternalServerError)
        }
    }

    newUser := &models.User{
        Email:     email,
        Fullname:  fullname,
        Username:  username,
        Telephone: telephone,
        IsSocial:  true,
        RoleID:    role.ID,
    }

    if err := a.authRepo.FacebookUserCreate(newUser); err != nil {
        log.Printf("Error creating user for email %s: %v", email, err)
        return nil, apiError.New(fmt.Sprintf("unable to create user: %v", err), http.StatusInternalServerError)
    }

    roleName := "user"
    accessToken, refreshToken, err := jwt.GenerateTokenPair(newUser.Email, a.Config.JWTSecret, newUser.AdminStatus, newUser.ID, roleName)
    if err != nil {
        log.Printf("Error generating token pair for user %s: %v", email, err)
        return nil, apiError.ErrInternalServerError
    }

    return &models.LoginResponse{
        UserResponse: models.UserResponse{
            ID:        newUser.ID,
            Fullname:  newUser.Fullname,
            Username:  newUser.Username,
            Telephone: newUser.Telephone,
            Email:     newUser.Email,
            RoleName:  roleName,
        },
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}