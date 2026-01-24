package eip712

import (
	"context"
	"errors"
	"time"
)

const (
	// DefaultTimestampTolerance is the default allowed time drift for signatures
	DefaultTimestampTolerance = 5 * time.Minute
)

// WalletVerificationMessage represents the EIP-712 typed data message
type WalletVerificationMessage struct {
	Wallet    string `json:"wallet"`
	Nonce     string `json:"nonce"`
	Timestamp int64  `json:"timestamp"`
}

// Config holds EIP-712 domain configuration
type Config struct {
	ChainID            int64
	VerifyingContract  string
	TimestampTolerance time.Duration
}

// Verifier defines the interface for EIP-712 signature verification
type Verifier interface {
	// VerifyWalletOwnership verifies wallet ownership using EIP-712 signature
	// Includes nonce reservation, timestamp validation, and signature verification
	VerifyWalletOwnership(ctx context.Context, address string, message WalletVerificationMessage, signature []byte) error

	// VerifySignatureOnly verifies only the cryptographic signature without nonce handling
	// Used for testing or when nonce is managed externally
	VerifySignatureOnly(address string, message WalletVerificationMessage, signature []byte) (bool, error)
}

// Error definitions
var (
	ErrInvalidSignature     = errors.New("invalid signature")
	ErrSignatureExpired     = errors.New("signature timestamp expired")
	ErrSignatureFuture      = errors.New("signature timestamp is in the future")
	ErrInvalidAddress       = errors.New("invalid ethereum address")
	ErrAddressMismatch      = errors.New("recovered address does not match")
	ErrInvalidSignatureLen  = errors.New("signature must be 65 bytes")
)
