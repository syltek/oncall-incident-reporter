package errors

import (
	"fmt"
	"net/http"
)

// AppError defines a custom error type with categorization and error wrapping
type AppError struct {
	Code       int    // HTTP status code
	Message    string // Default user-friendly error message
	Category   string // Error category: "client" or "server"
	WrappedErr error  // Optional wrapped error
}

// Constants for error categories
const (
	CategoryServer = "server"
	CategoryClient = "client"
)

// Predefined Errors
var (
	ErrBadRequest     = New(http.StatusBadRequest, "Bad request", "client", nil)
	ErrUnauthorized   = New(http.StatusUnauthorized, "Unauthorized", "client", nil)
	ErrForbidden      = New(http.StatusForbidden, "Forbidden", "client", nil)
	ErrInternalServer = New(http.StatusInternalServerError, "Internal server error", "server", nil)
)

// Error implements the error interface
func (e *AppError) Error() string {
	if e.WrappedErr != nil {
		return fmt.Sprintf("%d (%s): %s - %v", e.Code, e.Category, e.Message, e.WrappedErr)
	}
	return fmt.Sprintf("%d (%s): %s", e.Code, e.Category, e.Message)
}

// New creates a new AppError
func New(code int, message string, category string, wrappedErr error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Category:   category,
		WrappedErr: wrappedErr,
	}
}
