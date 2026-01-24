package nonce

import (
	"context"
	"errors"
	"time"
)

const (
	// DefaultTTL is the default nonce validity duration
	DefaultTTL = 5 * time.Minute
)

// Store defines the interface for nonce storage
// Implementations can use Redis, in-memory, or other backends
type Store interface {
	// Reserve attempts to reserve a nonce for use
	// Returns ErrNonceAlreadyUsed if nonce is already used or reserved
	Reserve(ctx context.Context, nonce, address string) error

	// MarkUsed marks a reserved nonce as used (after successful verification)
	MarkUsed(ctx context.Context, nonce, address string) error

	// Release releases a reserved nonce (on verification failure, allows retry)
	Release(ctx context.Context, nonce, address string) error
}

// Error definitions
var (
	ErrNonceAlreadyUsed = errors.New("nonce already used or reserved")
	ErrNonceNotFound    = errors.New("nonce not found")
)
