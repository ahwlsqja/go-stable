package errors

import (
	"fmt"
	"net/http"
)

// Error codes
const (
	// 4xx Client Errors
	CodeInvalidInput        = "INVALID_INPUT"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeIdempotencyConflict = "IDEMPOTENCY_CONFLICT"
	CodeInsufficientBalance = "INSUFFICIENT_BALANCE"
	CodeInsufficientStock   = "INSUFFICIENT_STOCK"
	CodeInvalidState        = "INVALID_STATE_TRANSITION"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"

	// 5xx Server Errors
	CodeInternal     = "INTERNAL_ERROR"
	CodeDBError      = "DB_ERROR"
	CodeLockFailed   = "LOCK_FAILED"
	CodeChainError   = "CHAIN_ERROR"
	CodeChainTimeout = "CHAIN_TIMEOUT"
)

// AppError represents a structured application error
type AppError struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	StatusCode int               `json:"-"`
	Details    map[string]any    `json:"details,omitempty"`
	Err        error             `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithDetails(details map[string]any) *AppError {
	e.Details = details
	return e
}

func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// Error constructors

func InvalidInput(message string) *AppError {
	return &AppError{
		Code:       CodeInvalidInput,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func NotFound(resource string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
	}
}

func Conflict(message string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

func IdempotencyConflict() *AppError {
	return &AppError{
		Code:       CodeIdempotencyConflict,
		Message:    "Request with this idempotency key already processed",
		StatusCode: http.StatusConflict,
	}
}

func InsufficientBalance(available, requested string) *AppError {
	return &AppError{
		Code:       CodeInsufficientBalance,
		Message:    fmt.Sprintf("Available balance %s is less than requested %s", available, requested),
		StatusCode: http.StatusBadRequest,
		Details: map[string]any{
			"available": available,
			"requested": requested,
		},
	}
}

func InsufficientStock(available, requested int64) *AppError {
	return &AppError{
		Code:       CodeInsufficientStock,
		Message:    fmt.Sprintf("Available stock %d is less than requested %d", available, requested),
		StatusCode: http.StatusBadRequest,
		Details: map[string]any{
			"available": available,
			"requested": requested,
		},
	}
}

func InvalidStateTransition(from, to string) *AppError {
	return &AppError{
		Code:       CodeInvalidState,
		Message:    fmt.Sprintf("Cannot transition from %s to %s", from, to),
		StatusCode: http.StatusBadRequest,
		Details: map[string]any{
			"from": from,
			"to":   to,
		},
	}
}

func Unauthorized(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

func Forbidden(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

func Internal(message string) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
	}
}

func DBError(err error) *AppError {
	return &AppError{
		Code:       CodeDBError,
		Message:    "Database error occurred",
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

func LockFailed(resource string) *AppError {
	return &AppError{
		Code:       CodeLockFailed,
		Message:    fmt.Sprintf("Failed to acquire lock for %s", resource),
		StatusCode: http.StatusConflict,
	}
}

func ChainError(message string) *AppError {
	return &AppError{
		Code:       CodeChainError,
		Message:    message,
		StatusCode: http.StatusServiceUnavailable,
	}
}
