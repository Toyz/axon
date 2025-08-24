package axon

import (
	"fmt"
	"net/http"
)

// HttpError represents an HTTP error with a specific status code and message
type HttpError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Details    any    `json:"details,omitempty"`
}

// Error implements the error interface
func (e *HttpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// NewHttpError creates a new HttpError with the given status code and message
func NewHttpError(statusCode int, message string) *HttpError {
	return &HttpError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// NewHttpErrorWithDetails creates a new HttpError with additional details
func NewHttpErrorWithDetails(statusCode int, message string, details any) *HttpError {
	return &HttpError{
		StatusCode: statusCode,
		Message:    message,
		Details:    details,
	}
}

// Common HTTP error constructors for convenience

// ErrBadRequest creates a 400 Bad Request error
func ErrBadRequest(message string) *HttpError {
	return NewHttpError(http.StatusBadRequest, message)
}

// ErrBadRequestWithDetails creates a 400 Bad Request error with details
func ErrBadRequestWithDetails(message string, details any) *HttpError {
	return NewHttpErrorWithDetails(http.StatusBadRequest, message, details)
}

// ErrUnauthorized creates a 401 Unauthorized error
func ErrUnauthorized(message string) *HttpError {
	return NewHttpError(http.StatusUnauthorized, message)
}

// ErrForbidden creates a 403 Forbidden error
func ErrForbidden(message string) *HttpError {
	return NewHttpError(http.StatusForbidden, message)
}

// ErrNotFound creates a 404 Not Found error
func ErrNotFound(message string) *HttpError {
	return NewHttpError(http.StatusNotFound, message)
}

// ErrConflict creates a 409 Conflict error
func ErrConflict(message string) *HttpError {
	return NewHttpError(http.StatusConflict, message)
}

// ErrUnprocessableEntity creates a 422 Unprocessable Entity error
func ErrUnprocessableEntity(message string) *HttpError {
	return NewHttpError(http.StatusUnprocessableEntity, message)
}

// ErrUnprocessableEntityWithDetails creates a 422 Unprocessable Entity error with validation details
func ErrUnprocessableEntityWithDetails(message string, details any) *HttpError {
	return NewHttpErrorWithDetails(http.StatusUnprocessableEntity, message, details)
}

// ErrInternalServerError creates a 500 Internal Server Error
func ErrInternalServerError(message string) *HttpError {
	return NewHttpError(http.StatusInternalServerError, message)
}