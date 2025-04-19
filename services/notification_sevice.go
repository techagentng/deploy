package services

import (
	"fmt"
	"github.com/oliveroneill/exponent-server-sdk-golang/sdk"

)

type NotificationService struct {
	expoClient *expo.PushClient
}

// convertToMapString converts a map[string]interface{} to map[string]string.
func convertToMapString(input map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range input {
		if strValue, ok := value.(string); ok {
			result[key] = strValue
		} else {
			result[key] = fmt.Sprintf("%v", value)
		}
	}
	return result
}

// isValidExpoPushToken validates if a given token is a valid Expo push token.
func isValidExpoPushToken(token string) bool {
	return len(token) > 0 && token[:2] == "ExponentPushToken"
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		expoClient: expo.NewPushClient(nil),
	}
}

func (s *NotificationService) SendPushNotification(
	userExpoPushToken string,
	title string,
	body string,
	data map[string]interface{},
) error {
	if !isValidExpoPushToken(userExpoPushToken) {
		return fmt.Errorf("invalid Expo push token")
	}

	message := &expo.PushMessage{
		To:       []expo.ExponentPushToken{expo.ExponentPushToken(userExpoPushToken)},
		Sound:    "default",
		Title:    title,
		Body:     body,
		Data:     convertToMapString(data),
		Priority: expo.DefaultPriority,
	}

	response, err := s.expoClient.Publish(message)
	if err != nil {
		return err
	}

	if response.ValidateResponse() != nil {
		return response.ValidateResponse()
	}

	return nil
}