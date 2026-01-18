package handler

import (
	"net/http"

	"github.com/ahwlsqja/go-stable/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

// ErrorResponse is the standard error response format
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// RespondError sends an error response
func RespondError(c *gin.Context, err error) {
	requestID, _ := c.Get("request_id")
	reqIDStr, _ := requestID.(string)

	if appErr, ok := errors.AsAppError(err); ok {
		c.JSON(appErr.HTTPStatus, ErrorResponse{
			Error: ErrorBody{
				Code:      appErr.Code,
				Message:   appErr.Message,
				RequestID: reqIDStr,
				Details:   appErr.Details,
			},
		})
		return
	}

	// Unknown error - treat as internal server error
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorBody{
			Code:      errors.ErrCodeInternal,
			Message:   "an unexpected error occurred",
			RequestID: reqIDStr,
		},
	})
}

// RespondSuccess sends a success response
func RespondSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// RespondCreated sends a 201 response
func RespondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}
