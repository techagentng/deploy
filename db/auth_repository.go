package db

import (
	"fmt"
	"log"
	"strings"
	"github.com/pkg/errors"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

type AuthRepository interface {
	CreateUser(user *models.User) (*models.User, error)
	CreateGoogleUser(user *models.CreateSocialUserParams) (*models.CreateSocialUserParams, error)
	IsEmailExist(email string) error
	IsPhoneExist(email string) error
	FindUserByUsername(username string) (*models.User, error)
	FindUserByEmail(email string) (*models.User, error)
	UpdateUser(user *models.User) error
	AddToBlackList(blacklist *models.Blacklist) error
	TokenInBlacklist(token string) bool
	VerifyEmail(email string, token string) error
	IsTokenInBlacklist(token string) bool
	UpdatePassword(password string, email string) error
	FindUserByID(id uint) (*models.User, error)
	// UpdateUserImage(user *models.User) error
	EditUserProfile(userID uint, userDetails *models.EditProfileResponse) error
	FindUserByMacAddress(macAddress string) (*models.LoginRequestMacAddress, error)
	ResetPassword(userID, NewPassword string) error
	CreateUserWithMacAddress(user *models.LoginRequestMacAddress) (*models.LoginRequestMacAddress, error)
	UpdateUserStatus(user *models.User) error
	UpdateUserOnlineStatus(user *models.User) error
	SetUserOffline(user *models.User) error
	GetOnlineUserCount() (int64, error)
	GetAllUsers() ([]models.User, error)
	UpsertUserImage(userID uint, filepath string) error
	FindRoleByID(roleID uuid.UUID) (*models.Role, error)
	FindRoleByUserEmail(email string) (*models.Role, error)
	FindRoleByName(name string) (*models.Role, error)
	GetUserRoleByUserID(userID uint) (*models.Role, error)
}

type authRepo struct {
	DB *gorm.DB
}

func NewAuthRepo(db *GormDB) AuthRepository {
	return &authRepo{db.DB}
}

func generateIDx2() uuid.UUID {
    id, err := uuid.NewUUID()
    if err != nil {
        log.Fatalf("Failed to generate UUID: %v", err)
    }
    return id
}

func (a *authRepo) CreateUser(user *models.User) (*models.User, error) {
    if user == nil {
        log.Println("CreateUser error: user is nil")
        return nil, errors.New("user is nil")
    }

    // Optional: Assign a default role if the Role is not set
    if user.RoleID == uuid.Nil {
        // Fetch or create the default role
        var defaultRole models.Role
        if err := a.DB.Where("name = ?", models.RoleUser).First(&defaultRole).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                // Role not found, create it
                defaultRole = models.Role{
                    ID:   generateIDx2(), // Use the updated function
                    Name: models.RoleUser,
                }
                if err := a.DB.Create(&defaultRole).Error; err != nil {
                    log.Printf("CreateUser error creating default role: %v", err)
                    return nil, err
                }
            } else {
                log.Printf("CreateUser error fetching default role: %v", err)
                return nil, err
            }
        }
        user.RoleID = defaultRole.ID
    }

    // Create the user in the database
    result := a.DB.Create(user)
    if result.Error != nil {
        log.Printf("CreateUser error: %v", result.Error)
        return nil, result.Error
    }

    // Return the created user
    return user, nil
}



// CreateUserWithMacAddress updates the MAC address field for an existing user or creates a new user with the provided MAC address
func (a *authRepo) CreateUserWithMacAddress(user *models.LoginRequestMacAddress) (*models.LoginRequestMacAddress, error) {
	// Attempt to find an existing user with the same MAC address
	existingUser := &models.User{}
	err := a.DB.Where("mac_address = ?", user.MacAddress).First(existingUser).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Println("DB error:", err)
		return nil, fmt.Errorf("could not find existing user: %v", err)
	}

	// If no existing user is found, create a new user with the provided MAC address
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = a.DB.Create(user).Error
		if err != nil {
			log.Println("DB error:", err)
			return nil, fmt.Errorf("could not create user: %v", err)
		}
	}

	return user, nil
}

func (a *authRepo) CreateGoogleUser(user *models.CreateSocialUserParams) (*models.CreateSocialUserParams, error) {
	err := a.DB.Create(user).Error
	if err != nil {
		return nil, fmt.Errorf("could not create user: %v", err)
	}
	return user, nil
}

func (a *authRepo) FindUserByUsername(username string) (*models.User, error) {
	db := a.DB
	user := &models.User{}
	err := db.Where("email = ? OR username = ?", username, username).First(user).Error
	if err != nil {
		return nil, fmt.Errorf("could not find user: %v", err)
	}
	return user, nil
}

func (a *authRepo) IsEmailExist(email string) error {
	var count int64
	err := a.DB.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No user found with this email, return nil
			return nil
		}
		// Return wrapped error for other errors
		return errors.Wrap(err, "gorm count error")
	}
	if count > 0 {
		// Email already exists, return specific error
		return errors.New("email already in use")
	}
	return nil
}

func (a *authRepo) IsPhoneExist(phone string) error {
	var count int64
	err := a.DB.Model(&models.User{}).Where("telephone = ?", phone).Count(&count).Error
	if err != nil {
		return errors.Wrap(err, "gorm.count error")
	}
	if count > 0 {
		return fmt.Errorf("phone number already in use")
	}
	return nil
}

func (a *authRepo) FindUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := a.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error finding user by email: %w", err)
	}
	return &user, nil
}

func (a *authRepo) UpdateUser(user *models.User) error {
	return nil
}

func (a *authRepo) AddToBlackList(blacklist *models.Blacklist) error {
	result := a.DB.Create(blacklist)
	return result.Error
}

func (a *authRepo) TokenInBlacklist(token string) bool {
	result := a.DB.Where("token = ?", token).Find(&models.Blacklist{})
	return result.Error != nil
}

func (a *authRepo) VerifyEmail(email string, token string) error {
	err := a.DB.Model(&models.User{}).Where("email = ?", email).Updates(models.User{IsEmailActive: true}).Error
	if err != nil {
		return err
	}

	err = a.AddToBlackList(&models.Blacklist{Token: token})
	return err
}

func normalizeToken(token string) string {
	// Trim leading and trailing white spaces
	return strings.TrimSpace(token)
}

func (a *authRepo) IsTokenInBlacklist(token string) bool {
	// Normalize the token
	normalizedToken := normalizeToken(token)

	var count int64
	// Assuming you have a Blacklist model with a Token field
	a.DB.Model(&models.Blacklist{}).Where("token = ?", normalizedToken).Count(&count)
	return count > 0
}

func (a *authRepo) UpdatePassword(password string, email string) error {
	err := a.DB.Model(&models.User{}).Where("email = ?", email).Updates(models.User{HashedPassword: password}).Error
	if err != nil {
		return err
	}
	return nil
}

func (a *authRepo) FindUserByID(id uint) (*models.User, error) {
	var user models.User
	err := a.DB.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (a *authRepo) FindUserByMacAddress(macAddress string) (*models.LoginRequestMacAddress, error) {
	var user models.LoginRequestMacAddress
	err := a.DB.Where("mac_address = ?", macAddress).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (a *authRepo) UpsertUserImage(userID uint, filepath string) error {
    var user models.User
    // Find the user by ID
    result := a.DB.Where("id = ?", userID).First(&user)

    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            // Record does not exist
            return errors.New("user not found")
        }
        // Other errors
        log.Printf("Error retrieving user record: %v", result.Error)
        return result.Error
    }

    // Update the user's thumbnail URL
    user.ThumbNailURL = filepath
    if err := a.DB.Save(&user).Error; err != nil {
        log.Printf("Error updating user thumbnail URL: %v", err)
        return err
    }

    return nil
}

// Repository method to update user profile in the database
func (a *authRepo) EditUserProfile(userID uint, userDetails *models.EditProfileResponse) error {
	// Fetch the user from the database
	var user models.User
	if err := a.DB.First(&user, userID).Error; err != nil {
		return err // return error if user not found
	}

	// Update user fields based on the userDetails
	if userDetails.FullName != "" {
		user.Fullname = userDetails.FullName
	}
	if userDetails.Username != "" {
		user.Username = userDetails.Username
	}
	// Update other fields as needed (e.g., profile image, email, etc.)

	// Perform the update operation
	if err := a.DB.Save(&user).Error; err != nil {
		return err
	}

	return nil
}

func (a *authRepo) ResetPassword(userID, NewPassword string) error {
	result := a.DB.Model(models.User{}).Where("id = ?", userID).Update("hashed_password", NewPassword)
	return result.Error
}

// Function in your repository to update the user's status
func (a *authRepo) UpdateUserStatus(user *models.User) error {
	return a.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("online", user.Online).Error
}

func (a *authRepo) UpdateUserOnlineStatus(user *models.User) error {
	log.Printf("Attempting to update user status: ID=%d, Online=%v", user.ID, user.Online)
	result := a.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("online", user.Online)
	if result.Error != nil {
		log.Printf("Error updating user status: %v", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Printf("No rows affected when updating user status for user ID: %d", user.ID)
		return fmt.Errorf("no rows affected")
	}
	log.Printf("Successfully updated user status for user ID: %d", user.ID)
	return nil
}

func (a *authRepo) SetUserOffline(user *models.User) error {
	log.Printf("Attempting to set user status to offline: ID=%d", user.ID)
	result := a.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("online", false)
	if result.Error != nil {
		log.Printf("Error setting user status to offline: %v", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Printf("No rows affected when setting user status to offline for user ID: %d", user.ID)
		return fmt.Errorf("no rows affected")
	}
	log.Printf("Successfully set user status to offline for user ID: %d", user.ID)
	return nil
}

func (a *authRepo) GetOnlineUserCount() (int64, error) {
	var count int64
	result := a.DB.Model(&models.User{}).Where("online = ?", true).Count(&count)
	if result.Error != nil {
		log.Printf("Error fetching online user count: %v", result.Error)
		return 0, result.Error
	}
	return count, nil
}

func (a *authRepo) GetAllUsers() ([]models.User, error) {
	var users []models.User
	result := a.DB.Find(&users)
	if result.Error != nil {
		log.Printf("Error fetching all users: %v", result.Error)
		return nil, result.Error
	}
	return users, nil
}

// FindRoleByID retrieves a role by its ID from the database.
func (r *authRepo) FindRoleByID(roleID uuid.UUID) (*models.Role, error) {
    var role *models.Role
    if err := r.DB.Where("id = ?", roleID).First(&role).Error; err != nil {
        return nil, err
    }
    return role, nil
}



// FindRoleByUserEmail retrieves the role of a user based on their email
func (a *authRepo) FindRoleByUserEmail(email string) (*models.Role, error) {
    // Define a User model to fetch the user's role by email
    var user models.User
    var role models.Role

    // Find the user by email
    if err := a.DB.Where("email = ?", email).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("user not found")
        }
        return nil, err
    }

    // Find the role by the user's RoleID
    if err := a.DB.Where("id = ?", user.Role).First(&role).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("role not found")
        }
        return nil, err
    }

    return &role, nil
}


// FindRoleByName fetches a role by its name from the database.
func (a *authRepo) FindRoleByName(name string) (*models.Role, error) {
    var role models.Role
    if err := a.DB.Where("name = ?", name).First(&role).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("Role not found:", name)
            return nil, errors.New("role not found")
        }
        return nil, err
    }
    return &role, nil
}


// GetUserRoleByUserID fetches the role associated with a given user ID.
func (a *authRepo) GetUserRoleByUserID(userID uint) (*models.Role, error) {
    // Define a variable to hold the user's role.
    var role models.Role

    // Query the database to find the role associated with the userID.
    // Assuming a join between the users and roles tables, or a direct relation.
    err := a.DB.Table("roles").
        Select("roles.*").
        Joins("JOIN user_roles ON user_roles.role_id = roles.id").
        Where("user_roles.user_id = ?", userID).
        First(&role).Error

    // Check if the role was found or if an error occurred during the query.
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            // If no role is found, return a nil role and a custom error.
            return nil, fmt.Errorf("no role found for user with ID %d", userID)
        }
        // For any other error, return it.
        return nil, err
    }

    // Return the role if found, otherwise return nil and an error.
    return &role, nil
}