package server

import (
	"context"
	"log"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/techagentng/citizenx/models"
)

// Save message to database and send a notification
func (s *Server) SendMessageHandler(c *gin.Context) {
    var msg models.Message
    if err := c.ShouldBindJSON(&msg); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request body"})
        return
    }

    // Extract SenderID from context as a string
    senderIDStr, exists := c.Get("userID")
    if !exists {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    // Convert SenderID to UUID
    senderID, err := uuid.Parse(senderIDStr.(string))
    if err != nil {
        c.JSON(400, gin.H{"error": "Invalid SenderID format"})
        return
    }
    msg.SenderID = senderID

    // Generate a new ConversationID if not provided
    if msg.ConversationID == uuid.Nil {
        // receiverID, err := s.PostRepository.FindReceiverIDBySomeLogic(senderID) // Define logic to get receiver
        // if err != nil {
        //     c.JSON(500, gin.H{"error": "Receiver not found"})
        //     return
        // }

        // conversationID, err := s.PostRepository.CreateNewConversation(senderID, receiverID)
        // if err != nil {
        //     c.JSON(500, gin.H{"error": "Failed to create conversation"})
        //     return
        // }
        // msg.ConversationID = conversationID
    }

    // Save the message
    if err := s.PostRepository.SaveMessage(&msg); err != nil {
        c.JSON(500, gin.H{"error": "Failed to save message"})
        return
    }

    // Fetch the receiver's device token
    // receiverID, err := s.PostRepository.FindReceiverIDByConversation(msg.ConversationID, senderID)
    // if err != nil {
    //     c.JSON(500, gin.H{"error": "Receiver not found"})
    //     return
    // }
    // _, err = s.PostRepository.GetReceiverDeviceToken(receiverID)
    // if err != nil {
    //     c.JSON(500, gin.H{"error": "Receiver device token not found"})
    //     return
    // }

    // Send push notification
    // if err := SendMessage(deviceToken, "New Message", msg.Content); err != nil {
    //     c.JSON(500, gin.H{"error": "Failed to send notification"})
    //     return
    // }

    // Update the conversation's last message and timestamp
    if err := s.PostRepository.UpdateConversationLastMessage(msg.ConversationID, msg.Content, time.Now()); err != nil {
        c.JSON(500, gin.H{"error": "Failed to update conversation"})
        return
    }

    // Send a success response
    c.JSON(200, gin.H{"message": "Message sent successfully"})
}

var messagingClient *messaging.Client

// SendMessage function to send a notification (Firebase Cloud Messaging)
func SendMessage(deviceToken, title, body string) error {
	message := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}

	// Send the message using Firebase Cloud Messaging
	_, err := messagingClient.Send(context.Background(), message)
	if err != nil {
		log.Println("Error sending message:", err)
		return err
	}

	log.Println("Message sent successfully")
	return nil
}




