package user

import (
	"time"

	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/repository/db"
)

// ============================================================================
// Request DTOs
// ============================================================================

// CreateUserRequest represents the request body for user registration
type CreateUserRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	Name  string `json:"name" binding:"required,min=2,max=100" example:"John Doe"`
	Phone string `json:"phone,omitempty" binding:"omitempty,min=10,max=20" example:"010-1234-5678"`
	Role  string `json:"role" binding:"required,oneof=BUYER SELLER BOTH" example:"BUYER"`
}

// UpdateUserProfileRequest represents the request body for profile update
type UpdateUserProfileRequest struct {
	Name  string `json:"name" binding:"required,min=2,max=100" example:"John Doe"`
	Phone string `json:"phone,omitempty" binding:"omitempty,min=10,max=20" example:"010-1234-5678"`
}

// UpdateUserRoleRequest represents the request body for role change
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=BUYER SELLER BOTH" example:"SELLER"`
}

// ListUsersRequest represents query parameters for listing users
type ListUsersRequest struct {
	Role      string `form:"role" binding:"omitempty,oneof=BUYER SELLER BOTH ADMIN"`
	KycStatus string `form:"kyc_status" binding:"omitempty,oneof=NONE PENDING VERIFIED REJECTED"`
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
}

// ============================================================================
// Response DTOs
// ============================================================================

// UserResponse represents the user data in API responses
type UserResponse struct {
	ID            string     `json:"id" example:"usr_abc123def456"`
	Email         string     `json:"email" example:"user@example.com"`
	Name          string     `json:"name" example:"John Doe"`
	Phone         string     `json:"phone,omitempty" example:"010-1234-5678"`
	Role          string     `json:"role" example:"BUYER"`
	KycStatus     string     `json:"kyc_status" example:"NONE"`
	KycVerifiedAt *time.Time `json:"kyc_verified_at,omitempty"`
	Status        string     `json:"status" example:"ACTIVE"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ListUsersResponse represents paginated user list
type ListUsersResponse struct {
	Users      []UserResponse `json:"users"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// ============================================================================
// Converters
// ============================================================================

// ToUserResponse converts db.User to UserResponse
func ToUserResponse(user *db.User) *UserResponse {
	if user == nil {
		return nil
	}

	response := &UserResponse{
		ID:        user.ExternalID.String,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		KycStatus: string(user.KycStatus),
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	if user.Phone.Valid {
		response.Phone = user.Phone.String
	}

	if user.KycVerifiedAt.Valid {
		response.KycVerifiedAt = &user.KycVerifiedAt.Time
	}

	return response
}

// ToUserResponseList converts []db.User to []UserResponse
func ToUserResponseList(users []db.User) []UserResponse {
	responses := make([]UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, *ToUserResponse(&user))
	}
	return responses
}
