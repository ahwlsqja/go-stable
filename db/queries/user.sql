-- ============================================================================
-- User Queries - Phase 1
-- ============================================================================

-- name: CreateUser :execresult
-- 사용자 생성 (external_id는 서비스 레이어에서 UUID 생성 후 전달)
INSERT INTO users (email, external_id, name, phone, role, status)
VALUES (?, ?, ?, ?, ?, 'ACTIVE');

-- name: GetUserByID :one
-- 소프트 삭제된 사용자 제외
SELECT * FROM users
WHERE id = ? AND status != 'DELETED';

-- name: GetUserByExternalID :one
-- 외부 식별자로 조회 (API 노출용, DELETED 제외)
SELECT * FROM users
WHERE external_id = ? AND status != 'DELETED';

-- name: GetUserByExternalIDIncludeDeleted :one
-- 내부 상태 전이 검증용 (DELETED 포함)
-- 멱등성 보장 및 상태 전이 체크에 사용
SELECT * FROM users
WHERE external_id = ?;

-- name: GetUserByEmail :one
-- 이메일로 조회 (중복 체크, 로그인 등)
SELECT * FROM users
WHERE email = ? AND status != 'DELETED';

-- name: GetUserForUpdate :one
-- 트랜잭션 내 row-lock (Primary 지갑 설정, 상태 변경 등 동시성 제어)
SELECT * FROM users
WHERE id = ? AND status != 'DELETED'
FOR UPDATE;

-- name: UpdateUserProfile :exec
-- 프로필 정보 업데이트 (이름, 전화번호)
UPDATE users
SET name = ?, phone = ?, updated_at = NOW()
WHERE id = ? AND status != 'DELETED';

-- name: UpdateUserRole :exec
-- 역할 변경 (BUYER, SELLER, BOTH, ADMIN)
UPDATE users
SET role = ?, updated_at = NOW()
WHERE id = ? AND status != 'DELETED';

-- ============================================================================
-- KYC 상태 변경 쿼리 (상태별 분리)
-- ============================================================================

-- name: UpdateUserKycToPending :exec
-- KYC 심사 요청 (NONE/REJECTED → PENDING)
UPDATE users
SET kyc_status = 'PENDING', updated_at = NOW()
WHERE id = ? AND status != 'DELETED';

-- name: UpdateUserKycToVerified :exec
-- KYC 승인 (PENDING → VERIFIED, kyc_verified_at 최초 설정 시만)
UPDATE users
SET kyc_status = 'VERIFIED',
    kyc_verified_at = COALESCE(kyc_verified_at, NOW()),
    updated_at = NOW()
WHERE id = ? AND status != 'DELETED';

-- name: UpdateUserKycToRejected :exec
-- KYC 거절 (PENDING → REJECTED)
UPDATE users
SET kyc_status = 'REJECTED', updated_at = NOW()
WHERE id = ? AND status != 'DELETED';

-- ============================================================================
-- 사용자 상태 변경 (:execresult로 RowsAffected 검증 가능)
-- ============================================================================

-- name: UpdateUserStatusToSuspended :execresult
-- 사용자 정지 (ACTIVE → SUSPENDED)
UPDATE users
SET status = 'SUSPENDED', updated_at = NOW()
WHERE id = ? AND status = 'ACTIVE';

-- name: UpdateUserStatusToActive :execresult
-- 정지 해제 (SUSPENDED → ACTIVE only, DELETED 복구 불가)
UPDATE users
SET status = 'ACTIVE', updated_at = NOW()
WHERE id = ? AND status = 'SUSPENDED';

-- name: UpdateUserStatusToDeleted :execresult
-- 소프트 삭제 (복구 불가, 단방향 전이)
UPDATE users
SET status = 'DELETED', updated_at = NOW()
WHERE id = ? AND status != 'DELETED';

-- ============================================================================
-- 목록 조회
-- ============================================================================

-- name: ListUsers :many
-- 사용자 목록 조회 (상태 필터 옵션, 페이징)
SELECT * FROM users
WHERE status != 'DELETED'
  AND (sqlc.narg('role') IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('kyc_status') IS NULL OR kyc_status = sqlc.narg('kyc_status'))
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountUsers :one
-- 사용자 수 조회 (페이징용)
SELECT COUNT(*) as total FROM users
WHERE status != 'DELETED'
  AND (sqlc.narg('role') IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('kyc_status') IS NULL OR kyc_status = sqlc.narg('kyc_status'));

-- name: ExistsUserByEmail :one
-- 이메일 중복 체크
SELECT EXISTS(
    SELECT 1 FROM users WHERE email = ? AND status != 'DELETED'
) as exists_flag;
