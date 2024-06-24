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
		respondWithMessage(c, http.StatusBadRequest, errors.GetUniqueContraintError(err), err.Error())
		return
	}

	if err, ok := err.(errors.ValidationError); ok {
		respondWithMessage(c, http.StatusBadRequest, errors.GetValidationError(err), err.Error())
		return
	}

	if e, ok := err.(*errors.Error); ok {
		respondWithMessage(c, e.Status, e, err.Error())
		return
	}

	respondWithMessage(c, http.StatusInternalServerError, &errors.Error{
		Message: err.Error(),
		Status:  http.StatusInternalServerError,
	}, err.Error())
}

func respondWithMessage(c *gin.Context, status int, e *errors.Error, message string) {
	responsedata := gin.H{
		"message": message,
		"data":    nil,
		"errors":  e.Message,
		"status":  status,
	}

	c.JSON(status, responsedata)
}

func InternalServerError(c *gin.Context) {
	respondWithMessage(c, http.StatusInternalServerError, &errors.Error{
		Message: "internal server error",
		Status:  http.StatusInternalServerError,
	}, "Internal Server Error")
}

func Unauthorized(c *gin.Context, message string) {
	respondWithMessage(c, http.StatusUnauthorized, &errors.Error{
		Message: message,
		Status:  http.StatusUnauthorized,
	}, "Unauthorized")
}
