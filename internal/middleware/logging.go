package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestID 미들웨어는 각 HTTP 요청에 대해 "요청 추적용 고유 ID"를 부여한다.
// - 클라이언트가 이미 X-Request-ID 헤더를 보내면 그 값을 그대로 사용한다(분산 추적/상위 시스템 연동).
// - 헤더가 없으면 서버가 UUID로 새 request_id를 생성한다.
// - 생성/확정된 request_id는
//   1) gin.Context의 key("request_id")에 저장되어 이후 핸들러/미들웨어에서 꺼내 쓸 수 있고
//   2) 응답 헤더 X-Request-ID로도 내려가서 클라이언트가 문제 발생 시 이 ID로 서버 로그를 추적할 수 있다.
// 주의: 이 미들웨어는 Logger보다 "먼저" 등록되어야 Logger가 request_id를 안전하게 사용할 수 있다.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) 먼저 클라이언트가 보내온 요청 ID가 있는지 확인한다.
		//    (예: API Gateway / Load Balancer / 프론트에서 이미 부여한 트레이싱 ID)
		requestID := c.GetHeader("X-Request-ID")

		// 2) 없다면 서버에서 새로 생성한다.
		//    UUID를 사용하면 충돌 가능성이 매우 낮고, 중앙 조정 없이도 고유성을 확보하기 쉽다.
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 3) 이후 처리 단계(핸들러/다른 미들웨어)에서 사용할 수 있도록 gin.Context에 저장한다.
		//    키 이름은 프로젝트 전체에서 "request_id"로 통일해야 한다.
		c.Set("request_id", requestID)

		// 4) 응답에도 동일한 request_id를 내려준다.
		//    클라이언트는 이 값을 가지고 "서버 로그에서 이 요청을 찾아달라" 요청할 수 있다.
		c.Header("X-Request-ID", requestID)

		// 5) 다음 미들웨어/핸들러로 흐름을 넘긴다.
		c.Next()
	}
}

// Logger 미들웨어는 요청/응답 정보를 구조화 로그(structured log)로 남긴다.
// - zap.Logger를 사용하여 JSON 기반의 필드 로그를 남기므로, ELK/Datadog/Cloud Logging 등에서 검색/집계가 쉽다.
// - 기록하는 핵심 정보:
//   * request_id: 특정 요청의 전체 흐름을 추적하기 위한 ID (RequestID 미들웨어가 먼저 실행되어야 함)
//   * method/path/query: 어떤 엔드포인트를 호출했는지
//   * status: 응답 HTTP 상태 코드
//   * latency: 요청 처리에 걸린 시간
//   * client_ip: 요청을 보낸 클라이언트 IP
//   * errors: gin 컨텍스트에 누적된 에러 목록(있을 때만)
// - 상태 코드에 따라 로그 레벨을 다르게 설정한다:
//   * 5xx: 서버 오류 -> Error
//   * 4xx: 클라이언트 오류 -> Warn
//   * 그 외: 정상 -> Info
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) 요청 시작 시간을 기록한다.
		//    c.Next() 이후에 time.Since(start)로 지연 시간(latency)을 계산한다.
		start := time.Now()

		// 2) 요청 URL 정보를 미리 잡아둔다.
		//    c.Next()를 지나고 나면 라우팅/리라이트/미들웨어에 의해 일부 값이 변할 수 있어,
		//    "원래 요청이 무엇이었는지" 보존하고 싶으면 초기에 저장하는 편이 안전하다.
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 3) 실제 요청 처리(다음 미들웨어/핸들러 실행)를 진행한다.
		//    여기서 핸들러가 실행되고 응답이 작성된 뒤 다시 이 지점으로 돌아온다.
		c.Next()

		// 4) 요청 처리에 걸린 시간(지연 시간)을 계산한다.
		latency := time.Since(start)

		// 5) 최종 응답 상태 코드를 가져온다.
		//    (핸들러에서 c.JSON, c.AbortWithStatus 등을 호출한 결과)
		statusCode := c.Writer.Status()

		// 6) RequestID 미들웨어가 저장한 request_id를 꺼낸다.
		//    request_id는 로그/에러 응답/트레이싱에서 공통 키로 쓰기 때문에 매우 중요하다.
		//    주의: RequestID 미들웨어가 먼저 실행되지 않으면 nil이거나 타입이 달라 panic이 날 수 있다.
		requestID, _ := c.Get("request_id")

		// 7) 구조화 로그에 붙일 필드들을 구성한다.
		//    zap.Field를 사용하면 각 값이 JSON 필드로 남아 검색/집계가 쉬워진다.
		fields := []zap.Field{
			// 요청 추적 ID: 동일 요청의 모든 로그를 이 값으로 묶을 수 있다.
			zap.String("request_id", requestID.(string)),

			// 어떤 HTTP 메서드로 호출했는지 (GET/POST/PUT/DELETE ...)
			zap.String("method", c.Request.Method),

			// 어떤 엔드포인트를 호출했는지 (경로)
			zap.String("path", path),

			// 쿼리스트링 (필요하면 파라미터 기반 분석 가능)
			zap.String("query", query),

			// 최종 HTTP 상태 코드
			zap.Int("status", statusCode),

			// 처리 지연 시간
			zap.Duration("latency", latency),

			// 클라이언트 IP (리버스 프록시 환경에선 X-Forwarded-For 영향을 받을 수 있음)
			zap.String("client_ip", c.ClientIP()),
		}

		// 8) Gin 컨텍스트에 에러가 누적되어 있으면 함께 기록한다.
		//    (예: c.Error(err)로 쌓인 에러들)
		//    에러가 항상 있는 게 아니므로 있을 때만 필드를 추가한다.
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// 9) 상태 코드에 따라 로그 레벨을 선택한다.
		//    - 5xx: 서버 문제이므로 Error 레벨로 남겨 알람/모니터링에 잡히게 한다.
		//    - 4xx: 클라이언트 요청 문제이므로 Warn으로 남긴다(필요 시 분석용).
		//    - 그 외(2xx/3xx): 정상 흐름이므로 Info로 남긴다.
		switch {
		case statusCode >= 500:
			logger.Error("server error", fields...)
		case statusCode >= 400:
			logger.Warn("client error", fields...)
		default:
			logger.Info("request", fields...)
		}
	}
}
