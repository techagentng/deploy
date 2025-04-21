package utils

import (
	"errors"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

// Secret key to sign tokens (Should be stored securely in env vars)
var jwtSecretKey = []byte(os.Getenv("JWT_SECRET"))

// TokenClaims represents the claims expected in the token
type TokenClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

// VerifyResetToken verifies the reset token, returning claims or an error
func VerifyResetToken(tokenString string) (*TokenClaims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	// Extract claims if token is valid
	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		// Check if the token has expired
		if claims.ExpiresAt < time.Now().Unix() {
			return nil, errors.New("token has expired")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GeneratePasswordResetToken generates a password reset token using the user's email
func GeneratePasswordResetToken(email string, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret key is missing")
	}

	// Create the JWT claims, including the user's email and expiration time for the reset token
	resetTokenClaims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 1).Unix(), // Token valid for 1 hour
		"type":  "password_reset_token",
	}

	// Create a new token object with the claims
	resetToken := jwt.NewWithClaims(jwt.SigningMethodHS256, resetTokenClaims)

	// Sign and get the complete encoded token as a string using the secret
	resetTokenString, err := resetToken.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return resetTokenString, nil
}

// HashPassword hashes the provided password using bcrypt
func HashPassword(password string) (string, error) {
	// bcrypt.DefaultCost is good for most cases (cost = 10)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plain password with its hashed version
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}