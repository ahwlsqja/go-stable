package wallet

import (
	"context"
	"database/sql"
	"encoding/hex"
	stderrors "errors"
	"strings"

	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/common/errors"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/repository/db"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/pkg/eip712"
	pkgdb "github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/pkg/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	mysqlErrDuplicateEntry = 1062
)

// Service handles wallet business logic
type Service struct {
	txRunner *pkgdb.TxRunner
	verifier eip712.Verifier
	logger   *zap.Logger
}

// NewService creates a new wallet service
func NewService(txRunner *pkgdb.TxRunner, verifier eip712.Verifier, logger *zap.Logger) *Service {
	return &Service{
		txRunner: txRunner,
		verifier: verifier,
		logger:   logger,
	}
}

// RegisterWallet registers a new wallet for a user
func (s *Service) RegisterWallet(ctx context.Context, userExternalID string, req *RegisterWalletRequest) (*db.Wallet, error) {
	// 1. Validate address format
	if err := ValidateEthereumAddress(req.Address); err != nil {
		return nil, err
	}

	// 2. Normalize address to lowercase
	address := strings.ToLower(req.Address)

	// 3. Get user by external ID
	user, err := s.txRunner.Queries().GetUserByExternalID(ctx, sql.NullString{String: userExternalID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("User")
		}
		s.logger.Error("failed to get user", zap.Error(err))
		return nil, errors.DBError(err)
	}

	// 4. Create wallet (UNIQUE 충돌 시 409로 처리)
	walletExternalID := uuid.New().String()
	label := sql.NullString{}
	if req.Label != "" {
		label = sql.NullString{String: req.Label, Valid: true}
	}

	result, err := s.txRunner.Queries().CreateWallet(ctx, db.CreateWalletParams{
		ExternalID: walletExternalID,
		UserID:     user.ID,
		Address:    address,
		Label:      label,
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, errors.Conflict("Wallet address already registered")
		}
		s.logger.Error("failed to create wallet", zap.Error(err))
		return nil, errors.DBError(err)
	}

	walletID, err := result.LastInsertId()
	if err != nil {
		return nil, errors.DBError(err)
	}

	// 5. Fetch and return created wallet
	wallet, err := s.txRunner.Queries().GetWalletByID(ctx, uint64(walletID))
	if err != nil {
		return nil, errors.DBError(err)
	}

	s.logger.Info("wallet registered",
		zap.String("wallet_external_id", walletExternalID),
		zap.String("address", address),
		zap.String("user_external_id", userExternalID),
	)

	return &wallet, nil
}

// GetWallet retrieves a wallet by external ID with ownership verification
func (s *Service) GetWallet(ctx context.Context, userExternalID, walletExternalID string) (*db.Wallet, error) {
	wallet, err := s.txRunner.Queries().GetWalletByExternalIDAndUser(ctx, db.GetWalletByExternalIDAndUserParams{
		ExternalID:   walletExternalID,
		ExternalID_2: sql.NullString{String: userExternalID, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("Wallet")
		}
		s.logger.Error("failed to get wallet", zap.Error(err))
		return nil, errors.DBError(err)
	}
	return &wallet, nil
}

// ListWallets retrieves all wallets for a user
func (s *Service) ListWallets(ctx context.Context, userExternalID string) (*ListWalletsResponse, error) {
	wallets, err := s.txRunner.Queries().ListWalletsByUserExternalID(ctx, sql.NullString{String: userExternalID, Valid: true})
	if err != nil {
		s.logger.Error("failed to list wallets", zap.Error(err))
		return nil, errors.DBError(err)
	}

	return &ListWalletsResponse{
		Wallets: ToWalletResponseList(wallets),
		Total:   int64(len(wallets)),
	}, nil
}

// UpdateLabel updates wallet label
func (s *Service) UpdateLabel(ctx context.Context, userExternalID, walletExternalID string, req *UpdateLabelRequest) (*db.Wallet, error) {
	// Get wallet with ownership check
	wallet, err := s.GetWallet(ctx, userExternalID, walletExternalID)
	if err != nil {
		return nil, err
	}

	// Update label
	result, err := s.txRunner.Queries().UpdateWalletLabel(ctx, db.UpdateWalletLabelParams{
		Label:  sql.NullString{String: req.Label, Valid: true},
		ID:     wallet.ID,
		UserID: wallet.UserID,
	})
	if err != nil {
		s.logger.Error("failed to update label", zap.Error(err))
		return nil, errors.DBError(err)
	}

	// Check RowsAffected
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, errors.NotFound("Wallet")
	}

	// Return updated wallet
	return s.GetWallet(ctx, userExternalID, walletExternalID)
}

// VerifyWallet verifies wallet ownership using EIP-712 signature
func (s *Service) VerifyWallet(ctx context.Context, userExternalID, walletExternalID string, req *VerifyWalletRequest) (*db.Wallet, error) {
	// 1. Parse signature
	signature, err := parseSignature(req.Signature)
	if err != nil {
		return nil, errors.InvalidInput("Invalid signature format")
	}

	// 2. Get wallet with ownership check
	wallet, err := s.GetWallet(ctx, userExternalID, walletExternalID)
	if err != nil {
		return nil, err
	}

	// 3. Already verified - idempotent success
	if wallet.IsVerified {
		return wallet, nil
	}

	// 4. Build verification message
	message := eip712.WalletVerificationMessage{
		Wallet:    wallet.Address,
		Nonce:     req.Message.Nonce,
		Timestamp: req.Message.Timestamp,
	}

	// 5. Verify signature (includes nonce + timestamp validation)
	if err := s.verifier.VerifyWalletOwnership(ctx, wallet.Address, message, signature); err != nil {
		s.logger.Warn("wallet verification failed",
			zap.String("wallet_external_id", walletExternalID),
			zap.String("address", wallet.Address),
			zap.Error(err),
		)
		// 외부 메시지는 고정, 상세는 로그로만
		return nil, errors.InvalidInput("Wallet verification failed")
	}

	// 6. Update wallet as verified + auto-set primary if first verified wallet
	return s.markWalletVerified(ctx, wallet)
}

// markWalletVerified marks wallet as verified and auto-sets as primary if needed
func (s *Service) markWalletVerified(ctx context.Context, wallet *db.Wallet) (*db.Wallet, error) {
	return pkgdb.WithTxResult(ctx, s.txRunner, func(q *db.Queries) (*db.Wallet, error) {
		// 1. Mark as verified
		result, err := q.UpdateWalletVerified(ctx, db.UpdateWalletVerifiedParams{
			ID:     wallet.ID,
			UserID: wallet.UserID,
		})
		if err != nil {
			s.logger.Error("failed to update wallet verified", zap.Error(err))
			return nil, errors.DBError(err)
		}

		affected, _ := result.RowsAffected()
		if affected == 0 {
			// Already verified - return current state
			w, err := q.GetWalletByID(ctx, wallet.ID)
			if err != nil {
				s.logger.Error("failed to get wallet after no-op verify", zap.Error(err))
				return nil, errors.DBError(err)
			}
			return &w, nil
		}

		// 2. Check if this is the first verified wallet (auto-set as primary)
		_, err = q.GetPrimaryWallet(ctx, wallet.UserID)
		if err != nil {
			if err == sql.ErrNoRows {
				// No primary wallet - set this one as primary
				s.logger.Info("auto-setting first verified wallet as primary",
					zap.Uint64("wallet_id", wallet.ID),
					zap.Uint64("user_id", wallet.UserID),
				)

				if err := s.setPrimaryInternal(q, wallet.ID, wallet.UserID); err != nil {
					// Primary 설정 실패는 치명적이지 않으나 로그 남김
					s.logger.Error("failed to auto-set primary wallet",
						zap.Uint64("wallet_id", wallet.ID),
						zap.Error(err),
					)
					// 검증 자체는 성공이므로 진행
				}
			} else {
				s.logger.Error("failed to check primary wallet", zap.Error(err))
			}
		}

		// 3. Return updated wallet
		updatedWallet, err := q.GetWalletByID(ctx, wallet.ID)
		if err != nil {
			s.logger.Error("failed to get updated wallet", zap.Error(err))
			return nil, errors.DBError(err)
		}

		s.logger.Info("wallet verified",
			zap.Uint64("wallet_id", wallet.ID),
			zap.String("address", wallet.Address),
		)

		return &updatedWallet, nil
	})
}

// setPrimaryInternal sets primary wallet within a transaction (internal helper)
func (s *Service) setPrimaryInternal(q *db.Queries, walletID, userID uint64) error {
	// Clear existing primary
	if err := q.ClearPrimaryWallet(context.Background(), userID); err != nil {
		return err
	}

	// Set as primary
	result, err := q.SetWalletPrimary(context.Background(), db.SetWalletPrimaryParams{
		ID:     walletID,
		UserID: userID,
	})
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.Internal("Failed to set wallet as primary")
	}

	// Update account primary wallet
	if err := q.UpdateAccountPrimaryWallet(context.Background(), db.UpdateAccountPrimaryWalletParams{
		PrimaryWalletID: sql.NullInt64{Int64: int64(walletID), Valid: true},
		OwnerID:         sql.NullInt64{Int64: int64(userID), Valid: true},
	}); err != nil {
		s.logger.Warn("failed to update account primary wallet", zap.Error(err))
	}

	return nil
}

// SetPrimary sets a wallet as the primary wallet
func (s *Service) SetPrimary(ctx context.Context, userExternalID, walletExternalID string) (*db.Wallet, error) {
	// Get wallet with ownership check
	wallet, err := s.GetWallet(ctx, userExternalID, walletExternalID)
	if err != nil {
		return nil, err
	}

	// Must be verified
	if !wallet.IsVerified {
		return nil, errors.InvalidInput("Wallet must be verified before setting as primary")
	}

	// Already primary - idempotent success
	if wallet.IsPrimary {
		return wallet, nil
	}

	// Get user for row-lock
	user, err := s.txRunner.Queries().GetUserByExternalID(ctx, sql.NullString{String: userExternalID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("User")
		}
		s.logger.Error("failed to get user for set primary", zap.Error(err))
		return nil, errors.DBError(err)
	}

	return pkgdb.WithTxResult(ctx, s.txRunner, func(q *db.Queries) (*db.Wallet, error) {
		// 1. Lock user row
		_, err := q.GetUserForUpdate(ctx, user.ID)
		if err != nil {
			s.logger.Error("failed to lock user row", zap.Error(err))
			return nil, errors.DBError(err)
		}

		// 2. Lock wallet row
		_, err = q.GetWalletForUpdate(ctx, db.GetWalletForUpdateParams{
			ID:     wallet.ID,
			UserID: wallet.UserID,
		})
		if err != nil {
			s.logger.Error("failed to lock wallet row", zap.Error(err))
			return nil, errors.DBError(err)
		}

		// 3. Clear existing primary
		if err := q.ClearPrimaryWallet(ctx, wallet.UserID); err != nil {
			s.logger.Error("failed to clear primary wallet", zap.Error(err))
			return nil, errors.DBError(err)
		}

		// 4. Set new primary
		result, err := q.SetWalletPrimary(ctx, db.SetWalletPrimaryParams{
			ID:     wallet.ID,
			UserID: wallet.UserID,
		})
		if err != nil {
			s.logger.Error("failed to set primary wallet", zap.Error(err))
			return nil, errors.DBError(err)
		}

		affected, _ := result.RowsAffected()
		if affected == 0 {
			return nil, errors.InvalidInput("Failed to set primary - wallet may not be verified")
		}

		// 5. Update account primary wallet
		if err := q.UpdateAccountPrimaryWallet(ctx, db.UpdateAccountPrimaryWalletParams{
			PrimaryWalletID: sql.NullInt64{Int64: int64(wallet.ID), Valid: true},
			OwnerID:         sql.NullInt64{Int64: int64(wallet.UserID), Valid: true},
		}); err != nil {
			s.logger.Error("failed to update account primary wallet", zap.Error(err))
			// 계속 진행 - 핵심은 wallet primary 설정
		}

		// 6. Return updated wallet
		updatedWallet, err := q.GetWalletByID(ctx, wallet.ID)
		if err != nil {
			s.logger.Error("failed to get updated wallet", zap.Error(err))
			return nil, errors.DBError(err)
		}

		s.logger.Info("wallet set as primary",
			zap.String("wallet_external_id", walletExternalID),
		)

		return &updatedWallet, nil
	})
}

// DeleteWallet deletes a wallet (soft delete)
func (s *Service) DeleteWallet(ctx context.Context, userExternalID, walletExternalID string) error {
	// Get wallet including deleted (for idempotency check)
	wallet, err := s.txRunner.Queries().GetWalletByExternalIDAndUserIncludeDeleted(ctx, db.GetWalletByExternalIDAndUserIncludeDeletedParams{
		ExternalID:   walletExternalID,
		ExternalID_2: sql.NullString{String: userExternalID, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NotFound("Wallet")
		}
		s.logger.Error("failed to get wallet for delete", zap.Error(err))
		return errors.DBError(err)
	}

	// Already deleted - idempotent success
	if wallet.DeletedAt.Valid {
		s.logger.Debug("wallet already deleted (idempotent)",
			zap.String("wallet_external_id", walletExternalID),
		)
		return nil
	}

	// Cannot delete primary wallet
	if wallet.IsPrimary {
		return errors.InvalidInput("Cannot delete primary wallet. Set another wallet as primary first.")
	}

	// Soft delete wallet
	result, err := s.txRunner.Queries().SoftDeleteWallet(ctx, db.SoftDeleteWalletParams{
		ID:     wallet.ID,
		UserID: wallet.UserID,
	})
	if err != nil {
		s.logger.Error("failed to delete wallet", zap.Error(err))
		return errors.DBError(err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		// Race condition: wallet became primary or was deleted by another process
		s.logger.Warn("wallet delete affected 0 rows",
			zap.String("wallet_external_id", walletExternalID),
			zap.Uint64("wallet_id", wallet.ID),
		)
		// Re-fetch to check current state
		currentWallet, fetchErr := s.txRunner.Queries().GetWalletByExternalIDAndUserIncludeDeleted(ctx, db.GetWalletByExternalIDAndUserIncludeDeletedParams{
			ExternalID:   walletExternalID,
			ExternalID_2: sql.NullString{String: userExternalID, Valid: true},
		})
		if fetchErr != nil {
			return errors.DBError(fetchErr)
		}
		if currentWallet.DeletedAt.Valid {
			// Already deleted by another process - idempotent success
			return nil
		}
		if currentWallet.IsPrimary {
			return errors.InvalidInput("Cannot delete wallet - it is now the primary wallet")
		}
		return errors.Internal("Failed to delete wallet")
	}

	s.logger.Info("wallet deleted",
		zap.String("wallet_external_id", walletExternalID),
	)

	return nil
}

// ============================================================================
// Helper functions
// ============================================================================

// ValidateEthereumAddress validates Ethereum address format
func ValidateEthereumAddress(address string) error {
	// Check basic format
	if !common.IsHexAddress(address) {
		return errors.InvalidInput("Invalid Ethereum address format")
	}

	// Check length (0x + 40 hex chars)
	if len(address) != 42 {
		return errors.InvalidInput("Invalid address length")
	}

	// Check checksum if mixed case (EIP-55)
	checksummed := common.HexToAddress(address).Hex()
	if address != strings.ToLower(address) && address != checksummed {
		return errors.InvalidInput("Invalid address checksum")
	}

	return nil
}

// parseSignature parses hex signature string to bytes
func parseSignature(sig string) ([]byte, error) {
	// Remove 0x prefix if present
	sig = strings.TrimPrefix(sig, "0x")

	// Must be 130 hex chars (65 bytes)
	if len(sig) != 130 {
		return nil, errors.InvalidInput("Signature must be 65 bytes")
	}

	return hex.DecodeString(sig)
}

// isDuplicateKeyError checks if the error is a MySQL duplicate key error
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if stderrors.As(err, &mysqlErr) {
		return mysqlErr.Number == mysqlErrDuplicateEntry
	}
	return strings.Contains(err.Error(), "Duplicate entry")
}
