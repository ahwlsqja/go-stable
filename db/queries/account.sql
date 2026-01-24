-- ============================================================================
-- Account Queries - Phase 1
-- ============================================================================
-- NOTE: external_id는 서비스 레이어에서 UUID 생성 후 전달

-- name: CreateAccount :execresult
-- 계정 생성 (사용자 회원가입 시 자동 생성)
-- account_type: USER(일반), MERCHANT(판매자), ESCROW(에스크로), SYSTEM(시스템)
INSERT INTO accounts (account_type, owner_id, external_id, status)
VALUES (?, ?, ?, 'ACTIVE');

-- name: GetAccountByID :one
-- ID로 계정 조회
SELECT * FROM accounts
WHERE id = ? AND status != 'CLOSED';

-- name: GetAccountByExternalID :one
-- 외부 식별자로 조회 (API 노출용)
SELECT * FROM accounts
WHERE external_id = ? AND status != 'CLOSED';

-- name: GetAccountByOwnerID :one
-- 사용자 ID로 계정 조회 (1:1 관계 가정)
SELECT * FROM accounts
WHERE owner_id = ? AND account_type = 'USER' AND status != 'CLOSED';

-- name: GetAccountForUpdate :one
-- 트랜잭션 내 row-lock (잔액 변경, Primary 지갑 연결 등)
SELECT * FROM accounts
WHERE id = ? AND status != 'CLOSED'
FOR UPDATE;

-- name: GetAccountByOwnerForUpdate :one
-- 사용자 ID로 계정 조회 + row-lock
SELECT * FROM accounts
WHERE owner_id = ? AND account_type = 'USER' AND status != 'CLOSED'
FOR UPDATE;

-- ============================================================================
-- 계정 업데이트
-- ============================================================================

-- name: UpdateAccountPrimaryWallet :exec
-- Primary 지갑 연결 (지갑 SetPrimary 후 호출)
UPDATE accounts
SET primary_wallet_id = ?, updated_at = NOW()
WHERE owner_id = ? AND account_type = 'USER' AND status != 'CLOSED';

-- name: ClearAccountPrimaryWallet :exec
-- Primary 지갑 연결 해제
UPDATE accounts
SET primary_wallet_id = NULL, updated_at = NOW()
WHERE owner_id = ? AND account_type = 'USER' AND status != 'CLOSED';

-- ============================================================================
-- 계정 상태 변경
-- ============================================================================

-- name: UpdateAccountStatusToSuspended :exec
-- 계정 정지 (ACTIVE → SUSPENDED)
UPDATE accounts
SET status = 'SUSPENDED', updated_at = NOW()
WHERE id = ? AND status = 'ACTIVE';

-- name: UpdateAccountStatusToActive :exec
-- 정지 해제 (SUSPENDED → ACTIVE)
UPDATE accounts
SET status = 'ACTIVE', updated_at = NOW()
WHERE id = ? AND status = 'SUSPENDED';

-- name: UpdateAccountStatusToClosed :exec
-- 계정 폐쇄 (사용자 탈퇴 시)
UPDATE accounts
SET status = 'CLOSED', updated_at = NOW()
WHERE id = ? AND status != 'CLOSED';

-- ============================================================================
-- 잔액 조회 (Phase 3+에서 사용, Phase 1에서는 미사용)
-- ============================================================================

-- name: GetAccountBalance :one
-- 잔액 조회 (balance, hold_balance, version)
SELECT id, balance, hold_balance, version
FROM accounts
WHERE id = ? AND status != 'CLOSED';

-- ============================================================================
-- 목록 조회
-- ============================================================================

-- name: ListAccountsByType :many
-- 타입별 계정 목록
SELECT * FROM accounts
WHERE account_type = ? AND status != 'CLOSED'
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountAccountsByType :one
-- 타입별 계정 수
SELECT COUNT(*) as total FROM accounts
WHERE account_type = ? AND status != 'CLOSED';
