package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/common/errors"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/repository/db"
	pkgdb "github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/pkg/db"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service handles user business logic
type Service struct {
	txRunner *pkgdb.TxRunner
	logger   *zap.Logger
}

// NewService creates a new user service
func NewService(txRunner *pkgdb.TxRunner, logger *zap.Logger) *Service {
	return &Service{
		txRunner: txRunner,
		logger:   logger,
	}
}

// CreateUser creates a new user with associated account
func (s *Service) CreateUser(ctx context.Context, req *CreateUserRequest) (*db.User, error) {
	// Check email uniqueness
	exists, err := s.txRunner.Queries().ExistsUserByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("failed to check email existence", zap.Error(err))
		return nil, errors.DBError(err)
	}
	if exists {
		return nil, errors.Conflict("Email already registered")
	}

	var createdUser *db.User

	// Transaction: Create user + Create account
	err = s.txRunner.WithTx(ctx, func(q *db.Queries) error {
		// 1. Create user
		userExternalID := uuid.New().String()
		phone := sql.NullString{}
		if req.Phone != "" {
			phone = sql.NullString{String: req.Phone, Valid: true}
		}

		result, err := q.CreateUser(ctx, db.CreateUserParams{
			Email:      req.Email,
			ExternalID: sql.NullString{String: userExternalID, Valid: true},
			Name:       req.Name,
			Phone:      phone,
			Role:       db.UsersRole(req.Role),
		})
		if err != nil {
			s.logger.Error("failed to create user", zap.Error(err))
			return errors.DBError(err)
		}

		userID, err := result.LastInsertId()
		if err != nil {
			return errors.DBError(err)
		}

		// 2. Create associated account (auto-creation on registration)
		accountExternalID := uuid.New().String()
		_, err = q.CreateAccount(ctx, db.CreateAccountParams{
			AccountType: db.AccountsAccountTypeUSER,
			OwnerID:     sql.NullInt64{Int64: userID, Valid: true},
			ExternalID:  sql.NullString{String: accountExternalID, Valid: true},
		})
		if err != nil {
			s.logger.Error("failed to create account", zap.Error(err))
			return errors.DBError(err)
		}

		// 3. Fetch created user
		user, err := q.GetUserByID(ctx, uint64(userID))
		if err != nil {
			return errors.DBError(err)
		}
		createdUser = &user

		s.logger.Info("user created",
			zap.String("external_id", userExternalID),
			zap.String("email", req.Email),
		)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

// GetUserByExternalID retrieves user by external ID
func (s *Service) GetUserByExternalID(ctx context.Context, externalID string) (*db.User, error) {
	user, err := s.txRunner.Queries().GetUserByExternalID(ctx, sql.NullString{String: externalID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("User")
		}
		s.logger.Error("failed to get user", zap.Error(err), zap.String("external_id", externalID))
		return nil, errors.DBError(err)
	}
	return &user, nil
}

// GetUserByID retrieves user by internal ID (for internal use)
func (s *Service) GetUserByID(ctx context.Context, id uint64) (*db.User, error) {
	user, err := s.txRunner.Queries().GetUserByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("User")
		}
		s.logger.Error("failed to get user", zap.Error(err), zap.Uint64("id", id))
		return nil, errors.DBError(err)
	}
	return &user, nil
}

// UpdateProfile updates user profile (name, phone)
func (s *Service) UpdateProfile(ctx context.Context, externalID string, req *UpdateUserProfileRequest) (*db.User, error) {
	// Get user first
	user, err := s.GetUserByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	phone := sql.NullString{}
	if req.Phone != "" {
		phone = sql.NullString{String: req.Phone, Valid: true}
	}

	err = s.txRunner.Queries().UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		Name:  req.Name,
		Phone: phone,
		ID:    user.ID,
	})
	if err != nil {
		s.logger.Error("failed to update profile", zap.Error(err), zap.String("external_id", externalID))
		return nil, errors.DBError(err)
	}

	// Return updated user
	return s.GetUserByExternalID(ctx, externalID)
}

// UpdateRole updates user role
func (s *Service) UpdateRole(ctx context.Context, externalID string, req *UpdateUserRoleRequest) (*db.User, error) {
	user, err := s.GetUserByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	// ADMIN role cannot be set via API
	if req.Role == "ADMIN" {
		return nil, errors.Forbidden("Cannot assign ADMIN role via API")
	}

	err = s.txRunner.Queries().UpdateUserRole(ctx, db.UpdateUserRoleParams{
		Role: db.UsersRole(req.Role),
		ID:   user.ID,
	})
	if err != nil {
		s.logger.Error("failed to update role", zap.Error(err), zap.String("external_id", externalID))
		return nil, errors.DBError(err)
	}

	return s.GetUserByExternalID(ctx, externalID)
}

// SuspendUser suspends a user (ACTIVE -> SUSPENDED)
func (s *Service) SuspendUser(ctx context.Context, externalID string) (*db.User, error) {
	user, err := s.GetUserByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	if user.Status != db.UsersStatusACTIVE {
		return nil, errors.InvalidStateTransition(string(user.Status), "SUSPENDED")
	}

	err = s.txRunner.Queries().UpdateUserStatusToSuspended(ctx, user.ID)
	if err != nil {
		s.logger.Error("failed to suspend user", zap.Error(err), zap.String("external_id", externalID))
		return nil, errors.DBError(err)
	}

	s.logger.Info("user suspended", zap.String("external_id", externalID))
	return s.GetUserByExternalID(ctx, externalID)
}

// ActivateUser reactivates a suspended user (SUSPENDED -> ACTIVE)
func (s *Service) ActivateUser(ctx context.Context, externalID string) (*db.User, error) {
	user, err := s.GetUserByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	// DELETED users cannot be reactivated (one-way transition)
	if user.Status == db.UsersStatusDELETED {
		return nil, errors.InvalidStateTransition("DELETED", "ACTIVE").
			WithDetails(map[string]any{"reason": "Deleted users cannot be reactivated"})
	}

	if user.Status != db.UsersStatusSUSPENDED {
		return nil, errors.InvalidStateTransition(string(user.Status), "ACTIVE")
	}

	err = s.txRunner.Queries().UpdateUserStatusToActive(ctx, user.ID)
	if err != nil {
		s.logger.Error("failed to activate user", zap.Error(err), zap.String("external_id", externalID))
		return nil, errors.DBError(err)
	}

	s.logger.Info("user activated", zap.String("external_id", externalID))
	return s.GetUserByExternalID(ctx, externalID)
}

// DeleteUser soft-deletes a user (one-way, cannot be recovered)
func (s *Service) DeleteUser(ctx context.Context, externalID string) error {
	user, err := s.GetUserByExternalID(ctx, externalID)
	if err != nil {
		return err
	}

	if user.Status == db.UsersStatusDELETED {
		return nil // Already deleted, idempotent
	}

	err = s.txRunner.WithTx(ctx, func(q *db.Queries) error {
		// 1. Delete user
		if err := q.UpdateUserStatusToDeleted(ctx, user.ID); err != nil {
			return errors.DBError(err)
		}

		// 2. Close associated account
		if err := q.UpdateAccountStatusToClosed(ctx, user.ID); err != nil {
			s.logger.Warn("failed to close account", zap.Error(err), zap.String("external_id", externalID))
			// Don't fail the whole operation for account closure
		}

		return nil
	})

	if err != nil {
		s.logger.Error("failed to delete user", zap.Error(err), zap.String("external_id", externalID))
		return err
	}

	s.logger.Info("user deleted", zap.String("external_id", externalID))
	return nil
}

// ListUsers retrieves paginated user list
func (s *Service) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	offset := (req.Page - 1) * req.PageSize

	// Build filter params
	params := db.ListUsersParams{
		Limit:  int32(req.PageSize),
		Offset: int32(offset),
	}
	countParams := db.CountUsersParams{}

	if req.Role != "" {
		params.Role = db.NullUsersRole{UsersRole: db.UsersRole(req.Role), Valid: true}
		countParams.Role = db.NullUsersRole{UsersRole: db.UsersRole(req.Role), Valid: true}
	}
	if req.KycStatus != "" {
		params.KycStatus = db.NullUsersKycStatus{UsersKycStatus: db.UsersKycStatus(req.KycStatus), Valid: true}
		countParams.KycStatus = db.NullUsersKycStatus{UsersKycStatus: db.UsersKycStatus(req.KycStatus), Valid: true}
	}

	// Get users
	users, err := s.txRunner.Queries().ListUsers(ctx, params)
	if err != nil {
		s.logger.Error("failed to list users", zap.Error(err))
		return nil, errors.DBError(err)
	}

	// Get total count
	total, err := s.txRunner.Queries().CountUsers(ctx, countParams)
	if err != nil {
		s.logger.Error("failed to count users", zap.Error(err))
		return nil, errors.DBError(err)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &ListUsersResponse{
		Users:      ToUserResponseList(users),
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ============================================================================
// KYC Operations (Admin only in production)
// ============================================================================

// RequestKycVerification requests KYC verification (NONE/REJECTED -> PENDING)
func (s *Service) RequestKycVerification(ctx context.Context, externalID string) (*db.User, error) {
	user, err := s.GetUserByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	if user.KycStatus != db.UsersKycStatusNONE && user.KycStatus != db.UsersKycStatusREJECTED {
		return nil, errors.InvalidStateTransition(string(user.KycStatus), "PENDING").
			WithDetails(map[string]any{
				"reason": fmt.Sprintf("KYC can only be requested from NONE or REJECTED status, current: %s", user.KycStatus),
			})
	}

	err = s.txRunner.Queries().UpdateUserKycToPending(ctx, user.ID)
	if err != nil {
		s.logger.Error("failed to request KYC", zap.Error(err), zap.String("external_id", externalID))
		return nil, errors.DBError(err)
	}

	s.logger.Info("KYC verification requested", zap.String("external_id", externalID))
	return s.GetUserByExternalID(ctx, externalID)
}

// ApproveKyc approves KYC (PENDING -> VERIFIED) - Admin only
func (s *Service) ApproveKyc(ctx context.Context, externalID string) (*db.User, error) {
	user, err := s.GetUserByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	if user.KycStatus != db.UsersKycStatusPENDING {
		return nil, errors.InvalidStateTransition(string(user.KycStatus), "VERIFIED")
	}

	err = s.txRunner.Queries().UpdateUserKycToVerified(ctx, user.ID)
	if err != nil {
		s.logger.Error("failed to approve KYC", zap.Error(err), zap.String("external_id", externalID))
		return nil, errors.DBError(err)
	}

	s.logger.Info("KYC approved", zap.String("external_id", externalID))
	return s.GetUserByExternalID(ctx, externalID)
}

// RejectKyc rejects KYC (PENDING -> REJECTED) - Admin only
func (s *Service) RejectKyc(ctx context.Context, externalID string) (*db.User, error) {
	user, err := s.GetUserByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	if user.KycStatus != db.UsersKycStatusPENDING {
		return nil, errors.InvalidStateTransition(string(user.KycStatus), "REJECTED")
	}

	err = s.txRunner.Queries().UpdateUserKycToRejected(ctx, user.ID)
	if err != nil {
		s.logger.Error("failed to reject KYC", zap.Error(err), zap.String("external_id", externalID))
		return nil, errors.DBError(err)
	}

	s.logger.Info("KYC rejected", zap.String("external_id", externalID))
	return s.GetUserByExternalID(ctx, externalID)
}
