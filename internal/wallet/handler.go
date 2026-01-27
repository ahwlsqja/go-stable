package wallet

import (
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/common/errors"
	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/common/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for wallet operations
type Handler struct {
	service *Service
}

// NewHandler creates a new wallet handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers wallet routes on the router group
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	// Wallet routes under /users/:id/wallets (uses :id to match user handler pattern)
	wallets := rg.Group("/users/:id/wallets")
	{
		wallets.POST("", h.RegisterWallet)
		wallets.GET("", h.ListWallets)
		wallets.GET("/:walletId", h.GetWallet)
		wallets.PUT("/:walletId/label", h.UpdateLabel)
		wallets.POST("/:walletId/verify", h.VerifyWallet)
		wallets.POST("/:walletId/set-primary", h.SetPrimary)
		wallets.DELETE("/:walletId", h.DeleteWallet)
	}
}

// validateUUID validates UUID format
func validateUUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.InvalidInput("Invalid UUID format")
	}
	return nil
}

// extractAndValidateUserID extracts and validates userId from path
func extractAndValidateUserID(c *gin.Context) (string, error) {
	userID := c.Param("userId")
	if err := validateUUID(userID); err != nil {
		return "", err
	}
	return userID, nil
}

// extractAndValidateWalletID extracts and validates walletId from path
func extractAndValidateWalletID(c *gin.Context) (string, error) {
	walletID := c.Param("walletId")
	if err := validateUUID(walletID); err != nil {
		return "", err
	}
	return walletID, nil
}

// RegisterWallet godoc
// @Summary Register a new wallet
// @Description Register a new Ethereum wallet for the user
// @Tags wallets
// @Accept json
// @Produce json
// @Param userId path string true "User external ID (UUID)"
// @Param request body RegisterWalletRequest true "Wallet registration data"
// @Success 201 {object} middleware.SuccessResponse{data=WalletResponse} "Wallet created"
// @Failure 400 {object} middleware.ErrorResponse "Invalid input"
// @Failure 404 {object} middleware.ErrorResponse "User not found"
// @Failure 409 {object} middleware.ErrorResponse "Wallet address already registered"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{userId}/wallets [post]
func (h *Handler) RegisterWallet(c *gin.Context) {
	userExternalID, err := extractAndValidateUserID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	var req RegisterWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, errors.InvalidInput(err.Error()))
		return
	}

	wallet, err := h.service.RegisterWallet(c.Request.Context(), userExternalID, &req)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondCreated(c, ToWalletResponse(wallet))
}

// GetWallet godoc
// @Summary Get wallet by ID
// @Description Retrieve wallet details by external ID
// @Tags wallets
// @Produce json
// @Param userId path string true "User external ID (UUID)"
// @Param walletId path string true "Wallet external ID (UUID)"
// @Success 200 {object} middleware.SuccessResponse{data=WalletResponse} "Wallet details"
// @Failure 400 {object} middleware.ErrorResponse "Invalid UUID format"
// @Failure 404 {object} middleware.ErrorResponse "Wallet not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{userId}/wallets/{walletId} [get]
func (h *Handler) GetWallet(c *gin.Context) {
	userExternalID, err := extractAndValidateUserID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}
	walletExternalID, err := extractAndValidateWalletID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	wallet, err := h.service.GetWallet(c.Request.Context(), userExternalID, walletExternalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToWalletResponse(wallet))
}

// ListWallets godoc
// @Summary List user wallets
// @Description Get all wallets for a user
// @Tags wallets
// @Produce json
// @Param userId path string true "User external ID (UUID)"
// @Success 200 {object} middleware.SuccessResponse{data=ListWalletsResponse} "Wallet list"
// @Failure 400 {object} middleware.ErrorResponse "Invalid UUID format"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{userId}/wallets [get]
// TODO: Phase 2+ - Add pagination (page, page_size) when wallet count grows
func (h *Handler) ListWallets(c *gin.Context) {
	userExternalID, err := extractAndValidateUserID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	result, err := h.service.ListWallets(c.Request.Context(), userExternalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, result)
}

// UpdateLabel godoc
// @Summary Update wallet label
// @Description Update the label of a wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Param userId path string true "User external ID (UUID)"
// @Param walletId path string true "Wallet external ID (UUID)"
// @Param request body UpdateLabelRequest true "Label update data"
// @Success 200 {object} middleware.SuccessResponse{data=WalletResponse} "Updated wallet"
// @Failure 400 {object} middleware.ErrorResponse "Invalid input"
// @Failure 404 {object} middleware.ErrorResponse "Wallet not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{userId}/wallets/{walletId}/label [put]
func (h *Handler) UpdateLabel(c *gin.Context) {
	userExternalID, err := extractAndValidateUserID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}
	walletExternalID, err := extractAndValidateWalletID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	var req UpdateLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, errors.InvalidInput(err.Error()))
		return
	}

	wallet, err := h.service.UpdateLabel(c.Request.Context(), userExternalID, walletExternalID, &req)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToWalletResponse(wallet))
}

// VerifyWallet godoc
// @Summary Verify wallet ownership
// @Description Verify wallet ownership using EIP-712 signature
// @Tags wallets
// @Accept json
// @Produce json
// @Param userId path string true "User external ID (UUID)"
// @Param walletId path string true "Wallet external ID (UUID)"
// @Param request body VerifyWalletRequest true "Signature and message data"
// @Success 200 {object} middleware.SuccessResponse{data=WalletResponse} "Verified wallet"
// @Failure 400 {object} middleware.ErrorResponse "Invalid signature or verification failed"
// @Failure 404 {object} middleware.ErrorResponse "Wallet not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{userId}/wallets/{walletId}/verify [post]
func (h *Handler) VerifyWallet(c *gin.Context) {
	userExternalID, err := extractAndValidateUserID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}
	walletExternalID, err := extractAndValidateWalletID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	var req VerifyWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, errors.InvalidInput(err.Error()))
		return
	}

	wallet, err := h.service.VerifyWallet(c.Request.Context(), userExternalID, walletExternalID, &req)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToWalletResponse(wallet))
}

// SetPrimary godoc
// @Summary Set wallet as primary
// @Description Set a verified wallet as the primary wallet
// @Tags wallets
// @Produce json
// @Param userId path string true "User external ID (UUID)"
// @Param walletId path string true "Wallet external ID (UUID)"
// @Success 200 {object} middleware.SuccessResponse{data=WalletResponse} "Primary wallet"
// @Failure 400 {object} middleware.ErrorResponse "Wallet not verified"
// @Failure 404 {object} middleware.ErrorResponse "Wallet not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{userId}/wallets/{walletId}/set-primary [post]
func (h *Handler) SetPrimary(c *gin.Context) {
	userExternalID, err := extractAndValidateUserID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}
	walletExternalID, err := extractAndValidateWalletID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	wallet, err := h.service.SetPrimary(c.Request.Context(), userExternalID, walletExternalID)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondOK(c, ToWalletResponse(wallet))
}

// DeleteWallet godoc
// @Summary Delete wallet
// @Description Delete a non-primary wallet (hard delete)
// @Tags wallets
// @Produce json
// @Param userId path string true "User external ID (UUID)"
// @Param walletId path string true "Wallet external ID (UUID)"
// @Success 204 "Wallet deleted"
// @Failure 400 {object} middleware.ErrorResponse "Cannot delete primary wallet"
// @Failure 404 {object} middleware.ErrorResponse "Wallet not found"
// @Failure 500 {object} middleware.ErrorResponse "Internal server error"
// @Router /api/v1/users/{userId}/wallets/{walletId} [delete]
func (h *Handler) DeleteWallet(c *gin.Context) {
	userExternalID, err := extractAndValidateUserID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}
	walletExternalID, err := extractAndValidateWalletID(c)
	if err != nil {
		middleware.RespondError(c, err)
		return
	}

	if err := h.service.DeleteWallet(c.Request.Context(), userExternalID, walletExternalID); err != nil {
		middleware.RespondError(c, err)
		return
	}

	middleware.RespondNoContent(c)
}
