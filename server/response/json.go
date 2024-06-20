package response

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/errors"
)

func JSON(c *gin.Context, message string, status int, data interface{}, err error) {
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}
	responsedata := gin.H{
		"message": message,
		"data":    data,
		"errors":  errMessage,
		"status":  http.StatusText(status),
	}

	c.JSON(status, responsedata)
}

func HandleErrors(c *gin.Context, err error) {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		respond(c, errors.GetUniqueContraintError(err))
		return
	}

	if err, ok := err.(errors.ValidationError); ok {
		respond(c, errors.GetValidationError(err))
		return
	}

	if e, ok := err.(*errors.Error); ok {
		respond(c, e)
		return
	}

	respond(c, &errors.Error{
		Message: err.Error(),
		Status:  http.StatusInternalServerError,
	})

}

func respond(c *gin.Context, e *errors.Error) {
	JSON(c, "", e.Status, nil, e)
}

func InternalServerError(c *gin.Context) {
	respond(c, errors.New("internal server error", http.StatusInternalServerError))
}

func Unauthorized(c *gin.Context, message string) {
	respond(c, errors.New(message, http.StatusUnauthorized))
}
