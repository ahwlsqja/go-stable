package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for request ID
	RequestIDKey = "request_id"
)

// RequestID middleware generates or extracts request ID for each request.
// If client sends X-Request-ID header, it uses that value.
// Otherwise, it generates a new UUID.
//
// Why:
// - 분산 환경에서 요청 추적 (로그, 에러, 모니터링)
// - 클라이언트가 제공하면 그대로 사용 → 클라이언트-서버 간 추적 연결
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in context for handlers/services to use
		c.Set(RequestIDKey, requestID)
		// Set in response header for client correlation
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID extracts request ID from gin context
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		return id.(string)
	}
	return ""
}
