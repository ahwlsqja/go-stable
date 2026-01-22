package handler

import (
	"net/http"

	"github.com/ahwlsqja/go-stable/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

// ErrorResponse는 API에서 사용하는 "표준 에러 응답 포맷"의 최상위 구조체이다.
// 모든 에러는 { "error": { ... } } 형태로 내려가도록 통일한다.
// 이렇게 감싸두면 성공 응답과 실패 응답의 스키마가 명확히 분리되고,
// 클라이언트 쪽에서도 error 객체만 파싱하면 되어서 처리 로직이 단순해진다.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody는 실제 에러 내용을 담는 구조체이다.
type ErrorBody struct {
	// Code: 기계가 분기 처리하기 위한 에러 코드 (예: VALIDATION_FAILED, NOT_FOUND, INTERNAL 등)
	Code string `json:"code"`

	// Message: 사람이 읽기 위한 에러 메시지 (UI 표시용)
	Message string `json:"message"`

	// RequestID: 이 요청을 식별하는 고유 ID (로그 추적용)
	// omitempty 이므로 값이 없으면 JSON 응답에서 필드 자체가 빠진다.
	RequestID string `json:"request_id,omitempty"`

	// Details: 필드별 검증 오류, 추가 디버깅 정보 등 구조화된 부가 정보
	// 예: { "email": "invalid format", "age": "must be >= 18" }
	Details map[string]any `json:"details,omitempty"`
}

// RespondError는 에러를 표준 JSON 포맷으로 변환하여 클라이언트에 응답하는 헬퍼 함수이다.
// 역할:
// 1) Gin Context에서 request_id를 꺼내고
// 2) err가 우리 서비스의 AppError 타입인지 판별한 뒤
// 3) 알맞은 HTTP 상태 코드와 에러 포맷으로 응답을 내려준다.
func RespondError(c *gin.Context, err error) {
	// RequestID 미들웨어에서 저장한 request_id를 가져온다.
	// 타입 단언이 실패할 수 있으므로 두 번째 반환값은 무시하고 string으로 캐스팅만 시도한다.
	requestID, _ := c.Get("request_id")
	reqIDStr, _ := requestID.(string)

	// err가 우리가 정의한 AppError 타입인지 검사한다.
	// AppError는 비즈니스/도메인 에러로, HTTP 상태 코드와 에러 코드, 메시지를 함께 가진다.
	if appErr, ok := errors.AsAppError(err); ok {
		c.JSON(appErr.HTTPStatus, ErrorResponse{
			Error: ErrorBody{
				Code:      appErr.Code,      // 도메인 에러 코드
				Message:   appErr.Message,   // 사용자에게 보여줄 메시지
				RequestID: reqIDStr,         // 요청 추적용 ID
				Details:   appErr.Details,   // 추가 상세 정보(선택)
			},
		})
		return
	}

	// 위에서 처리되지 않은 에러는 모두 "예상하지 못한 내부 서버 오류"로 취급한다.
	// 보안 및 안정성 관점에서 실제 에러 메시지는 노출하지 않고,
	// 고정된 메시지와 INTERNAL 에러 코드로 통일한다.
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorBody{
			Code:      errors.ErrCodeInternal,          // 공통 내부 오류 코드
			Message:   "an unexpected error occurred", // 클라이언트에 노출할 일반화된 메시지
			RequestID: reqIDStr,                       // 요청 추적용 ID
		},
	})
}

// RespondSuccess는 HTTP 200 OK와 함께 성공 응답 데이터를 내려주는 헬퍼 함수이다.
// 모든 핸들러에서 c.JSON(http.StatusOK, data)를 직접 호출하지 않고
// 이 함수를 통해 통일된 스타일로 응답하도록 유도한다.
func RespondSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// RespondCreated는 리소스 생성 성공 시 사용하는 헬퍼 함수이다.
// HTTP 201 Created 상태 코드와 함께 생성된 리소스 정보를 내려준다.
// 예: 회원가입, 주문 생성, 트랜잭션 생성 등.
func RespondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}
