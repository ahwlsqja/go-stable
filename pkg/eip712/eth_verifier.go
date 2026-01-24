package eip712

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/pkg/nonce"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"go.uber.org/zap"
)

// EthVerifier implements Verifier interface using go-ethereum
type EthVerifier struct {
	config     Config
	nonceStore nonce.Store
	typedData  apitypes.TypedData
	logger     *zap.Logger
}

// Compile-time interface compliance check
var _ Verifier = (*EthVerifier)(nil)

// NewEthVerifier creates a new EIP-712 verifier
func NewEthVerifier(config Config, nonceStore nonce.Store, logger *zap.Logger) *EthVerifier {
	if config.TimestampTolerance == 0 {
		config.TimestampTolerance = DefaultTimestampTolerance
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"WalletVerification": {
				{Name: "wallet", Type: "address"},
				{Name: "nonce", Type: "string"},
				{Name: "timestamp", Type: "uint256"},
			},
		},
		PrimaryType: "WalletVerification",
		Domain: apitypes.TypedDataDomain{
			Name:              "B2B Settlement",
			Version:           "1",
			ChainId:           (*apitypes.HexOrDecimal256)(big.NewInt(config.ChainID)),
			VerifyingContract: config.VerifyingContract,
		},
	}

	return &EthVerifier{
		config:     config,
		nonceStore: nonceStore,
		typedData:  typedData,
		logger:     logger,
	}
}

// VerifyWalletOwnership verifies wallet ownership with full nonce + timestamp handling
func (v *EthVerifier) VerifyWalletOwnership(
	ctx context.Context,
	address string,
	message WalletVerificationMessage,
	signature []byte,
) error {
	// 1. Validate address format
	if !common.IsHexAddress(address) {
		return ErrInvalidAddress
	}

	// 2. Validate timestamp (within tolerance)
	if err := v.validateTimestamp(message.Timestamp); err != nil {
		return err
	}

	// 3. Reserve nonce (prevents replay)
	if err := v.nonceStore.Reserve(ctx, message.Nonce, address); err != nil {
		v.logger.Warn("nonce reservation failed",
			zap.String("address", address),
			zap.String("nonce", message.Nonce),
			zap.Error(err),
		)
		return fmt.Errorf("nonce validation failed: %w", err)
	}

	// 4. Verify signature
	valid, err := v.VerifySignatureOnly(address, message, signature)
	if err != nil || !valid {
		// Release nonce on failure (allow retry with same nonce)
		if releaseErr := v.nonceStore.Release(ctx, message.Nonce, address); releaseErr != nil {
			v.logger.Error("failed to release nonce after verification failure",
				zap.String("address", address),
				zap.Error(releaseErr),
			)
		}
		if err != nil {
			return err
		}
		return ErrAddressMismatch
	}

	// 5. Mark nonce as used (successful verification)
	if err := v.nonceStore.MarkUsed(ctx, message.Nonce, address); err != nil {
		v.logger.Error("failed to mark nonce as used",
			zap.String("address", address),
			zap.Error(err),
		)
		// Don't fail the verification, just log
	}

	v.logger.Info("wallet ownership verified",
		zap.String("address", address),
	)
	return nil
}

// VerifySignatureOnly verifies only the cryptographic signature
func (v *EthVerifier) VerifySignatureOnly(
	address string,
	message WalletVerificationMessage,
	signature []byte,
) (bool, error) {
	if len(signature) != 65 {
		return false, ErrInvalidSignatureLen
	}

	// Build message map for hashing
	messageMap := map[string]interface{}{
		"wallet":    message.Wallet,
		"nonce":     message.Nonce,
		"timestamp": big.NewInt(message.Timestamp),
	}

	// 1. Compute domain separator hash
	domainSeparator, err := v.typedData.HashStruct("EIP712Domain", v.typedData.Domain.Map())
	if err != nil {
		return false, fmt.Errorf("failed to hash domain: %w", err)
	}

	// 2. Compute message hash
	messageHash, err := v.typedData.HashStruct("WalletVerification", messageMap)
	if err != nil {
		return false, fmt.Errorf("failed to hash message: %w", err)
	}

	// 3. Byte-level concatenation (NOT string concat!)
	// \x19\x01 + domainSeparator + messageHash
	rawData := make([]byte, 0, 66) // 2 + 32 + 32
	rawData = append(rawData, 0x19, 0x01)
	rawData = append(rawData, domainSeparator...)
	rawData = append(rawData, messageHash...)

	// 4. Keccak256 hash
	digest := crypto.Keccak256(rawData)

	// 5. Normalize v value (27/28 -> 0/1)
	sig := make([]byte, 65)
	copy(sig, signature)
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	// 6. Recover public key from signature
	pubKey, err := crypto.SigToPub(digest, sig)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// 7. Derive address from public key
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	// 8. Compare addresses (case-insensitive)
	return strings.EqualFold(recoveredAddr.Hex(), address), nil
}

// validateTimestamp checks if the timestamp is within acceptable range
func (v *EthVerifier) validateTimestamp(timestamp int64) error {
	msgTime := time.Unix(timestamp, 0)
	now := time.Now()

	// Check if too old
	if msgTime.Before(now.Add(-v.config.TimestampTolerance)) {
		return ErrSignatureExpired
	}

	// Check if too far in future
	if msgTime.After(now.Add(v.config.TimestampTolerance)) {
		return ErrSignatureFuture
	}

	return nil
}
