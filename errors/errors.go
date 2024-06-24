package errors

import (
	"errors"
	"fmt"
	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Error struct {
	Message string
	Status  int
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Respond(c *gin.Context) {
	responsedata := gin.H{
		"message": "",
		"data":    nil,
		"errors":  e.Error(),
		"status":  e.Status,
	}

	c.JSON(e.Status, responsedata)
}

func New(message string, status int) *Error {
	return &Error{
		Message: message,
		Status:  status,
	}
}

// InActiveUserError defines an inactive user error
var InActiveUserError = errors.New("user is inactive")
var ErrNotFound = New("not found", http.StatusNotFound)
var ErrInternalServerError = New("internal server error", http.StatusInternalServerError)
var ErrBadRequest = New("bad request", http.StatusBadRequest)
var ErrDuplicateRequest = New("entity already exist", http.StatusBadRequest)

//var ErrUnauthorized = New("unauthorized", http.StatusUnauthorized)

// InValidPasswordError
var ErrInvalidPassword = New("invalid email or password", http.StatusUnauthorized)

func GetUniqueContraintError(err error) *Error {
    fields := strings.Split(err.Error(), "UNIQUE constraint failed: ")
    if len(fields) < 2 {
        return &Error{
            Message: "email or telephone already exist",
            Status:  http.StatusBadRequest,
        }
    }
    return &Error{
        Message: fmt.Sprintf("%s must be unique", strings.Split(fields[1], ".")[1]),
        Status:  http.StatusBadRequest,
    }
}


func GetValidationError(err ValidationError) *Error {
	return &Error{
		Message: err.Error(),
		Status:  http.StatusBadRequest,
	}
}

func ErrorHandler(c *gin.Context, info ratelimit.Info) {
	c.String(429, "Too many requests. Try again in "+time.Until(info.ResetTime).String())
}
