package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// 헬스 체크 핸들러
type HealthHandler struct {
	db    *sql.DB
	redis *redis.Client
}

// DB와 Redis 의존성을 주입하여 HealthHandler를 생성한다.
func NewHealthHandler(db *sql.DB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ReadyResponse struct {
	DB    string `json:"db"`
	Redis string `json:"redis"`
	Chain string `json:"chain,omitempty"`
}

// Health returns basic health status (liveness probe)
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

// Ready checks all dependencies (readiness probe)
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := ReadyResponse{
		DB:    "ok",
		Redis: "ok",
	}
	statusCode := http.StatusOK

	// Check DB
	if err := h.db.PingContext(ctx); err != nil {
		response.DB = "error: " + err.Error()
		statusCode = http.StatusServiceUnavailable
	}

	// Check Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		response.Redis = "error: " + err.Error()
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}
