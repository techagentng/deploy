package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

// LikeRepository interface
type PostRepository interface {
	CreatePost(post *models.Post) error
	GetPostsByUserID(userID uint) ([]models.Post, error)
	GetAllPosts() ([]models.Post, error)
	GetPostByID(id string) (*models.Post, error)
	SaveMessage(msg interface{}) error
	FindReceiverIDByConversation(conversationID uuid.UUID, senderID uuid.UUID) (uuid.UUID, error)
	GetReceiverDeviceToken(receiverID uint) (string, error)
	UpdateConversationLastMessage(conversationID uuid.UUID, lastMessage string, updatedAt time.Time) error
	FindReceiverIDBySomeLogic(senderID uuid.UUID) (uuid.UUID, error)
	// CreateNewConversation(senderID uuid.UUID, receiverID uuid.UUID) (uuid.UUID, error)
}

// likeRepo struct
type postRepo struct {
	DB *gorm.DB
}

// NewLikeRepo creates a new instance of LikeRepository
func NewPostRepo(db *GormDB) PostRepository {
	return &postRepo{db.DB}
}

func (r *postRepo) CreatePost(post *models.Post) error {
	if err := r.DB.Create(post).Error; err != nil {
		return err
	}
	return nil
}

// GetPostsByUserID fetches all posts created by a specific user based on userID
func (r *postRepo) GetPostsByUserID(userID uint) ([]models.Post, error) {
	var posts []models.Post
	// Fetch posts where the userID matches
	err := r.DB.Where("user_id = ?", userID).Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *postRepo) GetAllPosts() ([]models.Post, error) {
	var posts []models.Post

	if err := r.DB.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *postRepo) GetPostByID(id string) (*models.Post, error) {
	var post models.Post
	if err := r.DB.Where("id = ?", id).First(&post).Error; err != nil {
		return nil, fmt.Errorf("error retrieving post with ID %s: %w", id, err)
	}
	return &post, nil
}

// SaveMessage saves a message to the database
func (r *postRepo) SaveMessage(msg interface{}) error {
	return r.DB.Create(&msg).Error
}

// FindReceiverIDByConversation finds the receiver ID from a conversation
func (r *postRepo) FindReceiverIDByConversation(conversationID uuid.UUID, senderID uuid.UUID) (uuid.UUID, error) {
	var receiverID uuid.UUID
	err := r.DB.Raw("SELECT user_id FROM conversation_participants WHERE conversation_id = ? AND user_id != ?", conversationID, senderID).Scan(&receiverID).Error
	if err != nil {
		return uuid.Nil, err
	}
	return receiverID, nil
}

// GetReceiverDeviceToken fetches the receiver's device token
func (r *postRepo) GetReceiverDeviceToken(receiverID uint) (string, error) {
	var deviceToken string
	err := r.DB.Raw("SELECT device_token FROM users WHERE id = ?", receiverID).Scan(&deviceToken).Error
	if err != nil {
		return "", err
	}
	return deviceToken, nil
}

func (r *postRepo) UpdateConversationLastMessage(conversationID uuid.UUID, lastMessage string, updatedAt time.Time) error {
	return r.DB.Model(&struct{ ConversationID uuid.UUID }{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message": lastMessage,
			"updated_at":   updatedAt,
		}).Error
}

func (r *postRepo) FindReceiverIDBySomeLogic(senderID uuid.UUID) (uuid.UUID, error) {
    var receiverID uuid.UUID

    // Example query: find the most recent conversation involving the sender
    query := `
        SELECT receiver_id
        FROM conversations
        WHERE sender_id = ?  -- Use ? placeholder for GORM-style queries
        ORDER BY updated_at DESC
        LIMIT 1;
    `
    
    // Execute the query using raw SQL (GORM supports raw SQL)
    result := r.DB.Raw(query, senderID).Scan(&receiverID)

    // If no record is found, return an appropriate error message
    if result.Error != nil {
        if result.Error == sql.ErrNoRows {
            return uuid.Nil, fmt.Errorf("no receiver found for sender: %v", senderID)
        }
        return uuid.Nil, fmt.Errorf("error fetching receiver_id: %v", result.Error)
    }

    return receiverID, nil
}

// CreateNewConversation creates a new conversation between the sender and receiver.
// func (r *postRepo) CreateNewConversation(senderID, receiverID uuid.UUID) (uuid.UUID, error) {
//     // Create a new conversation instance
//     conversation := models.Conversation{
//         Participants: []models.User{
//             {ID: senderID},
//             {ID: receiverID},
//         },
//         LastMessage: "",
//         CreatedAt:   time.Now(),
//         UpdatedAt:   time.Now(),
//     }

//     // Save the new conversation to the database
//     if err := r.DB.Create(&conversation).Error; err != nil {
//         return uuid.Nil, err
//     }

//     // Return the newly created conversation's ID
//     return conversation.ID, nil
// }


