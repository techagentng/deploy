package errors

import (
	"fmt"

	validator "github.com/go-playground/validator/v10"
)

// TODO: refactor this, might be needed in other places
// FieldError wraps around the validator error so it
// can be used and caught specifically
type FieldError struct {
	err validator.FieldError
}

func (q *FieldError) String() string {
	// var sb strings.Builder
	err := q.err

	return fmt.Sprintf("%s is invalid: %v", err.Field(), err.Value())

	// sb.WriteString("validation failed on field '" + err.Field() + "'")
	// sb.WriteString(", condition: " + err.ActualTag())

	// // Print condition parameters, e.g. oneof=red blue -> { red blue }
	// if err.Param() != "" {
	// 	sb.WriteString(" { " + err.Param() + " }")
	// }

	// if err.Value() != nil && err.Value() != "" {
	// 	sb.WriteString(fmt.Sprintf(", actual: %v", err.Value()))
	// }

	// return sb.String()
}

// NewFieldError returns a field error
func NewFieldError(err validator.FieldError) *FieldError {
	return &FieldError{err: err}
}

// ValidationError defines error that occur due to validation
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}
