package middleware

import (
	"net/http"

	"github.com/ahwlsqja/go-stable/internal/common/errors"
	"github.com/gin-gonic/gin"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains error details
type ErrorBody struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// SuccessResponse represents the standard success response format
type SuccessResponse struct {
	Data any `json:"data"`
}

// RespondSuccess sends a successful JSON response
func RespondSuccess(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, SuccessResponse{Data: data})
}

// RespondError sends an error JSON response
// Handles both *errors.AppError and generic errors
func RespondError(c *gin.Context, err error) {
	requestID := GetRequestID(c)

	var appErr *errors.AppError
	switch e := err.(type) {
	case *errors.AppError:
		appErr = e
	default:
		// Wrap unknown errors as internal error
		appErr = errors.Internal("An unexpected error occurred")
	}

	c.JSON(appErr.StatusCode, ErrorResponse{
		Error: ErrorBody{
			Code:      appErr.Code,
			Message:   appErr.Message,
			RequestID: requestID,
			Details:   appErr.Details,
		},
	})
}

// RespondCreated sends a 201 Created response
func RespondCreated(c *gin.Context, data any) {
	RespondSuccess(c, http.StatusCreated, data)
}

// RespondOK sends a 200 OK response
func RespondOK(c *gin.Context, data any) {
	RespondSuccess(c, http.StatusOK, data)
}

// RespondNoContent sends a 204 No Content response
func RespondNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
