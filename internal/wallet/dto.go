package wallet

import (
	"time"

	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/repository/db"
)

// ============================================================================
// Request DTOs
// ============================================================================

// RegisterWalletRequest represents the request body for wallet registration
// NOTE: Address 형식 검증은 서비스 레이어에서 ValidateEthereumAddress()로 수행
type RegisterWalletRequest struct {
	Address string `json:"address" binding:"required,len=42" example:"0x742d35Cc6634C0532925a3b844Bc454e4438f44e"`
	Label   string `json:"label,omitempty" binding:"omitempty,max=50" example:"My Main Wallet"`
}

// VerifyWalletRequest represents the request body for wallet verification
type VerifyWalletRequest struct {
	// Signature: 0x prefix + 130 hex chars (65 bytes)
	Signature string                     `json:"signature" binding:"required,len=132" example:"0x1234...abcd"`
	Message   VerifyWalletRequestMessage `json:"message" binding:"required"`
}

// VerifyWalletRequestMessage contains the EIP-712 message data
type VerifyWalletRequestMessage struct {
	Nonce     string `json:"nonce" binding:"required,min=8,max=64" example:"550e8400-e29b-41d4-a716-446655440000"`
	Timestamp int64  `json:"timestamp" binding:"required,gt=0" example:"1706000000"`
}

// UpdateLabelRequest represents the request body for label update
type UpdateLabelRequest struct {
	Label string `json:"label" binding:"required,max=50" example:"Trading Wallet"`
}

// ============================================================================
// Response DTOs
// ============================================================================

// WalletResponse represents the wallet data in API responses
type WalletResponse struct {
	ID         string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Address    string    `json:"address" example:"0x742d35cc6634c0532925a3b844bc454e4438f44e"`
	Label      string    `json:"label,omitempty" example:"My Main Wallet"`
	IsPrimary  bool      `json:"is_primary" example:"false"`
	IsVerified bool      `json:"is_verified" example:"false"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ListWalletsResponse represents the wallet list response
type ListWalletsResponse struct {
	Wallets []WalletResponse `json:"wallets"`
	Total   int64            `json:"total"`
}

// ============================================================================
// Converters
// ============================================================================

// ToWalletResponse converts db.Wallet to WalletResponse
func ToWalletResponse(wallet *db.Wallet) *WalletResponse {
	if wallet == nil {
		return nil
	}

	response := &WalletResponse{
		ID:         wallet.ExternalID,
		Address:    wallet.Address,
		IsPrimary:  wallet.IsPrimary,
		IsVerified: wallet.IsVerified,
		CreatedAt:  wallet.CreatedAt,
		UpdatedAt:  wallet.UpdatedAt,
	}

	if wallet.Label.Valid {
		response.Label = wallet.Label.String
	}

	return response
}

// ToWalletResponseList converts []db.Wallet to []WalletResponse
func ToWalletResponseList(wallets []db.Wallet) []WalletResponse {
	responses := make([]WalletResponse, 0, len(wallets))
	for _, wallet := range wallets {
		responses = append(responses, *ToWalletResponse(&wallet))
	}
	return responses
}
