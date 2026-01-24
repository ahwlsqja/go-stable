package user

import (
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/common/errors"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/common/middleware"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for user operations
type Handler struct {
	service *Service
}

// NewHandler creates a new user handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers user routes on the router group
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.POST("", h.CreateUser)
		users.GET("", h.ListUsers)
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateProfile)
		users.PUT("/:id/role", h.UpdateRole)
		users.POST("/:id/suspend", h.SuspendUser)
		users.POST("/:id/activate", h.ActivateUser)
		users.DELETE("/:id", h.DeleteUser)

		// KYC endpoints
		users.POST("/:id/kyc/request", h.RequestKyc)
		users.POST("/:id/kyc/approve", h.ApproveKyc)   // TODO: Phase 6 - Add admin auth check
		users.POST("/:id/kyc/reject", h.RejectKyc)    // TODO: Phase 6 - Add admin auth check
	}
}

// CreateUser godoc
// @Summary Create a new user
// @Description Register a new user with email, name, and role. An account is automatically created.
// @Tags users
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User registration data"
// @Success 201 {object} middleware.SuccessResponse{data=UserResponse} "User created successfully"
// @Failure 400 {object} middleware.ErrorResponse "Invalid input"
// @Failure 409 {object} middleware.ErrorResponse "Email already registered"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, errors.InvalidInput(err.Error()))
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), &req)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondCreated(c, ToUserResponse(user))
}

// GetUser godoc
// @Summary Get user by ID
// @Description Retrieve user details by external ID
// @Tags users
// @Produce json
// @Param id path string true "User external ID"
// @Success 200 {object} middleware.SuccessResponse{data=UserResponse} "User details"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	externalID := c.Param("id")

	user, err := h.service.GetUserByExternalID(c.Request.Context(), externalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToUserResponse(user))
}

// ListUsers godoc
// @Summary List users
// @Description Get paginated list of users with optional filters
// @Tags users
// @Produce json
// @Param role query string false "Filter by role" Enums(BUYER, SELLER, BOTH, ADMIN)
// @Param kyc_status query string false "Filter by KYC status" Enums(NONE, PENDING, VERIFIED, REJECTED)
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} middleware.SuccessResponse{data=ListUsersResponse} "User list"
// @Failure 400 {object} middleware.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	var req ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		middleware.RespondError(c, errors.InvalidInput(err.Error()))
		return
	}

	// Set defaults if not provided
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	result, err := h.service.ListUsers(c.Request.Context(), &req)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, result)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update user's name and phone number
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User external ID"
// @Param request body UpdateUserProfileRequest true "Profile update data"
// @Success 200 {object} middleware.SuccessResponse{data=UserResponse} "Updated user"
// @Failure 400 {object} middleware.ErrorResponse "Invalid input"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{id} [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	externalID := c.Param("id")

	var req UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, errors.InvalidInput(err.Error()))
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), externalID, &req)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToUserResponse(user))
}

// UpdateRole godoc
// @Summary Update user role
// @Description Change user's role (BUYER, SELLER, BOTH)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User external ID"
// @Param request body UpdateUserRoleRequest true "Role update data"
// @Success 200 {object} middleware.SuccessResponse{data=UserResponse} "Updated user"
// @Failure 400 {object} middleware.ErrorResponse "Invalid input"
// @Failure 403 {object} middleware.ErrorResponse "Cannot assign ADMIN role"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{id}/role [put]
func (h *Handler) UpdateRole(c *gin.Context) {
	externalID := c.Param("id")

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, errors.InvalidInput(err.Error()))
		return
	}

	user, err := h.service.UpdateRole(c.Request.Context(), externalID, &req)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToUserResponse(user))
}

// SuspendUser godoc
// @Summary Suspend user
// @Description Suspend an active user (ACTIVE -> SUSPENDED)
// @Tags users
// @Produce json
// @Param id path string true "User external ID"
// @Success 200 {object} middleware.SuccessResponse{data=UserResponse} "Suspended user"
// @Failure 400 {object} middleware.ErrorResponse "Invalid state transition"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{id}/suspend [post]
func (h *Handler) SuspendUser(c *gin.Context) {
	externalID := c.Param("id")

	user, err := h.service.SuspendUser(c.Request.Context(), externalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToUserResponse(user))
}

// ActivateUser godoc
// @Summary Activate user
// @Description Reactivate a suspended user (SUSPENDED -> ACTIVE)
// @Tags users
// @Produce json
// @Param id path string true "User external ID"
// @Success 200 {object} middleware.SuccessResponse{data=UserResponse} "Activated user"
// @Failure 400 {object} middleware.ErrorResponse "Invalid state transition (deleted users cannot be reactivated)"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{id}/activate [post]
func (h *Handler) ActivateUser(c *gin.Context) {
	externalID := c.Param("id")

	user, err := h.service.ActivateUser(c.Request.Context(), externalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToUserResponse(user))
}

// DeleteUser godoc
// @Summary Delete user
// @Description Soft-delete a user (irreversible)
// @Tags users
// @Produce json
// @Param id path string true "User external ID"
// @Success 204 "User deleted"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	externalID := c.Param("id")

	if err := h.service.DeleteUser(c.Request.Context(), externalID); err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondNoContent(c)
}

// RequestKyc godoc
// @Summary Request KYC verification
// @Description Request KYC verification for the user (NONE/REJECTED -> PENDING)
// @Tags users
// @Produce json
// @Param id path string true "User external ID"
// @Success 200 {object} middleware.SuccessResponse{data=UserResponse} "KYC requested"
// @Failure 400 {object} middleware.ErrorResponse "Invalid state transition"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{id}/kyc/request [post]
func (h *Handler) RequestKyc(c *gin.Context) {
	externalID := c.Param("id")

	user, err := h.service.RequestKycVerification(c.Request.Context(), externalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToUserResponse(user))
}

// ApproveKyc godoc
// @Summary Approve KYC
// @Description Approve user's KYC verification (PENDING -> VERIFIED) - Admin only
// @Tags users
// @Produce json
// @Param id path string true "User external ID"
// @Success 200 {object} middleware.SuccessResponse{data=UserResponse} "KYC approved"
// @Failure 400 {object} middleware.ErrorResponse "Invalid state transition"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/users/{id}/kyc/approve [post]
func (h *Handler) ApproveKyc(c *gin.Context) {
	// TODO: Phase 6 - Add admin role verification via JWT claims
	externalID := c.Param("id")

	user, err := h.service.ApproveKyc(c.Request.Context(), externalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToUserResponse(user))
}

// RejectKyc godoc
// @Summary Reject KYC
// @Description Reject user's KYC verification (PENDING -> REJECTED) - Admin only
// @Tags users
// @Produce json
// @Param id path string true "User external ID"
// @Success 200 {object} middleware.SuccessResponse{data=UserResponse} "KYC rejected"
// @Failure 400 {object} middleware.ErrorResponse "Invalid state transition"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/users/{id}/kyc/reject [post]
func (h *Handler) RejectKyc(c *gin.Context) {
	// TODO: Phase 6 - Add admin role verification via JWT claims
	externalID := c.Param("id")

	user, err := h.service.RejectKyc(c.Request.Context(), externalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToUserResponse(user))
}
