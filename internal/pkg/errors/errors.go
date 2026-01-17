package errors

import (
	"fmt"
	"net/http"
)

// Error codes
const (
	// 4xx Client Errors
	ErrCodeInvalidInput        = "INVALID_INPUT"
	ErrCodeIdempotencyConflict = "IDEMPOTENCY_CONFLICT"
	ErrCodeInsufficientBalance = "INSUFFICIENT_BALANCE"
	ErrCodePaymentExpired      = "PAYMENT_EXPIRED"
	ErrCodeInvalidState        = "INVALID_STATE_TRANSITION"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"

	// 5xx Server Errors
	ErrCodeInternal      = "INTERNAL_ERROR"
	ErrCodeDBError       = "DB_ERROR"
	ErrCodeLockFailed    = "LOCK_ACQUISITION_FAILED"
	ErrCodeChainError    = "CHAIN_RPC_ERROR"
	ErrCodeChainTimeout  = "CHAIN_CONFIRMATION_TIMEOUT"
	ErrCodeRedisError    = "REDIS_ERROR"
)

// AppError represents a structured application error
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	HTTPStatus int                    `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
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

func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// Error constructors

func NewInvalidInput(message string) *AppError {
	return &AppError{
		Code:       ErrCodeInvalidInput,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

func NewIdempotencyConflict(key string) *AppError {
	return &AppError{
		Code:       ErrCodeIdempotencyConflict,
		Message:    "request with this idempotency key already processed",
		HTTPStatus: http.StatusConflict,
		Details:    map[string]interface{}{"idempotency_key": key},
	}
}

func NewInsufficientBalance(available, requested string) *AppError {
	return &AppError{
		Code:       ErrCodeInsufficientBalance,
		Message:    fmt.Sprintf("available balance %s is less than requested %s", available, requested),
		HTTPStatus: http.StatusUnprocessableEntity,
		Details: map[string]interface{}{
			"available": available,
			"requested": requested,
		},
	}
}

func NewPaymentExpired(paymentID uint64) *AppError {
	return &AppError{
		Code:       ErrCodePaymentExpired,
		Message:    "payment authorization has expired",
		HTTPStatus: http.StatusUnprocessableEntity,
		Details:    map[string]interface{}{"payment_id": paymentID},
	}
}

func NewInvalidStateTransition(from, to string) *AppError {
	return &AppError{
		Code:       ErrCodeInvalidState,
		Message:    fmt.Sprintf("cannot transition from %s to %s", from, to),
		HTTPStatus: http.StatusUnprocessableEntity,
		Details: map[string]interface{}{
			"current_state": from,
			"target_state":  to,
		},
	}
}

func NewNotFound(resource string, id interface{}) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
		Details: map[string]interface{}{
			"resource": resource,
			"id":       id,
		},
	}
}

func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

func NewForbidden(message string) *AppError {
	return &AppError{
		Code:       ErrCodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

func NewInternalError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeInternal,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

func NewDBError(err error) *AppError {
	return &AppError{
		Code:       ErrCodeDBError,
		Message:    "database operation failed",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

func NewLockFailed(resource string) *AppError {
	return &AppError{
		Code:       ErrCodeLockFailed,
		Message:    fmt.Sprintf("failed to acquire lock for %s", resource),
		HTTPStatus: http.StatusServiceUnavailable,
		Details:    map[string]interface{}{"resource": resource},
	}
}

func NewChainError(err error) *AppError {
	return &AppError{
		Code:       ErrCodeChainError,
		Message:    "blockchain RPC operation failed",
		HTTPStatus: http.StatusBadGateway,
		Err:        err,
	}
}

func NewChainTimeout(txHash string) *AppError {
	return &AppError{
		Code:       ErrCodeChainTimeout,
		Message:    "transaction confirmation timeout",
		HTTPStatus: http.StatusGatewayTimeout,
		Details:    map[string]interface{}{"tx_hash": txHash},
	}
}

func NewRedisError(err error) *AppError {
	return &AppError{
		Code:       ErrCodeRedisError,
		Message:    "redis operation failed",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError converts an error to AppError if possible
func AsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}
