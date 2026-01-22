package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db  *sql.DB
	rdb *redis.Client
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *sql.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:  db,
		rdb: rdb,
	}
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// ReadyResponse represents readiness check response
type ReadyResponse struct {
	Status string `json:"status" example:"ok"`
	DB     string `json:"db" example:"ok"`
	Redis  string `json:"redis" example:"ok"`
}

// Health godoc
// @Summary Health check
// @Description Returns server health status
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

// Ready godoc
// @Summary Readiness check
// @Description Returns server readiness status including DB and Redis connectivity
// @Tags health
// @Produce json
// @Success 200 {object} ReadyResponse
// @Failure 503 {object} ReadyResponse
// @Router /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	response := ReadyResponse{
		Status: "ok",
		DB:     "ok",
		Redis:  "ok",
	}
	statusCode := http.StatusOK

	// Check DB
	if err := h.db.PingContext(ctx); err != nil {
		response.DB = "error"
		response.Status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	// Check Redis
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		response.Redis = "error"
		response.Status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}
