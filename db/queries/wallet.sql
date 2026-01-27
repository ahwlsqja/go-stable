-- ============================================================================
-- Wallet Queries - Phase 1
-- ============================================================================
-- NOTE: wallet address는 저장/조회 시 lower-case normalize 적용
--       서비스 레이어에서 strings.ToLower() 처리 후 쿼리 호출
-- NOTE: Soft Delete 적용 - deleted_at IS NULL 조건 필수

-- name: CreateWallet :execresult
-- 지갑 등록 (address는 서비스에서 lower-case 변환 후 전달)
-- is_verified=false, is_primary=false 기본값
INSERT INTO wallets (external_id, user_id, address, label, is_primary, is_verified)
VALUES (?, ?, ?, ?, false, false);

-- name: GetWalletByID :one
-- ID로 지갑 조회 (내부 전용 - 삭제된 지갑 제외)
SELECT * FROM wallets WHERE id = ? AND deleted_at IS NULL;

-- name: GetWalletByExternalID :one
-- 외부 식별자로 지갑 조회 (삭제된 지갑 제외)
SELECT * FROM wallets WHERE external_id = ? AND deleted_at IS NULL;

-- name: GetWalletByExternalIDIncludeDeleted :one
-- 외부 식별자로 지갑 조회 (삭제된 지갑 포함 - 멱등성 체크용)
SELECT * FROM wallets WHERE external_id = ?;

-- name: GetWalletByExternalIDAndUser :one
-- 외부 식별자 + 사용자 소유권 검증 조회 (외부 API용, 삭제 제외)
SELECT w.* FROM wallets w
JOIN users u ON w.user_id = u.id
WHERE w.external_id = ? AND u.external_id = ? AND w.deleted_at IS NULL;

-- name: GetWalletByExternalIDAndUserIncludeDeleted :one
-- 외부 식별자 + 사용자 소유권 검증 조회 (멱등성 체크용, 삭제 포함)
SELECT w.* FROM wallets w
JOIN users u ON w.user_id = u.id
WHERE w.external_id = ? AND u.external_id = ?;

-- name: GetWalletByIDAndUser :one
-- ID + 사용자 소유권 검증 조회 (내부용, 삭제 제외)
SELECT * FROM wallets
WHERE id = ? AND user_id = ? AND deleted_at IS NULL;

-- name: GetWalletByAddress :one
-- 주소로 지갑 조회 (address는 lower-case로 전달, 삭제 제외)
SELECT * FROM wallets WHERE address = ? AND deleted_at IS NULL;

-- name: GetWalletForUpdate :one
-- 트랜잭션 내 row-lock (검증, Primary 설정 등)
SELECT * FROM wallets
WHERE id = ? AND user_id = ? AND deleted_at IS NULL
FOR UPDATE;

-- name: GetPrimaryWallet :one
-- 사용자의 Primary 지갑 조회 (삭제 제외)
SELECT * FROM wallets
WHERE user_id = ? AND is_primary = true AND deleted_at IS NULL
LIMIT 1;

-- name: ListWalletsByUser :many
-- 사용자의 전체 지갑 목록 (삭제 제외)
SELECT * FROM wallets
WHERE user_id = ? AND deleted_at IS NULL
ORDER BY is_primary DESC, created_at ASC;

-- name: ListWalletsByUserExternalID :many
-- 사용자 external_id로 지갑 목록 조회 (외부 API용, 삭제 제외)
SELECT w.* FROM wallets w
JOIN users u ON w.user_id = u.id
WHERE u.external_id = ? AND w.deleted_at IS NULL
ORDER BY w.is_primary DESC, w.created_at ASC;

-- name: CountWalletsByUser :one
-- 사용자의 지갑 수 조회 (삭제 제외)
SELECT COUNT(*) as total FROM wallets
WHERE user_id = ? AND deleted_at IS NULL;

-- ============================================================================
-- 지갑 상태 업데이트 (:execresult로 RowsAffected 검증 가능)
-- ============================================================================

-- name: UpdateWalletVerified :execresult
-- EIP-712 서명 검증 완료 (삭제되지 않은 지갑만)
UPDATE wallets
SET is_verified = true, updated_at = NOW()
WHERE id = ? AND user_id = ? AND is_verified = false AND deleted_at IS NULL;

-- name: UpdateWalletLabel :execresult
-- 지갑 라벨 변경 (삭제되지 않은 지갑만)
UPDATE wallets
SET label = ?, updated_at = NOW()
WHERE id = ? AND user_id = ? AND deleted_at IS NULL;

-- ============================================================================
-- Primary 지갑 설정 (트랜잭션 내 호출)
-- ============================================================================

-- name: ClearPrimaryWallet :exec
-- 기존 Primary 지갑 해제 (SetPrimary 트랜잭션 첫 단계, 삭제 제외)
UPDATE wallets
SET is_primary = false, updated_at = NOW()
WHERE user_id = ? AND is_primary = true AND deleted_at IS NULL;

-- name: SetWalletPrimary :execresult
-- 새 Primary 지갑 설정 (소유권 + 검증 상태 확인, 삭제 제외)
-- SetPrimary 트랜잭션: 1) GetUserForUpdate 2) ClearPrimaryWallet 3) SetWalletPrimary
UPDATE wallets
SET is_primary = true, updated_at = NOW()
WHERE id = ? AND user_id = ? AND is_verified = true AND deleted_at IS NULL;

-- ============================================================================
-- 지갑 삭제 (Soft Delete)
-- ============================================================================

-- name: SoftDeleteWallet :execresult
-- Soft Delete - deleted_at 설정
-- Primary 지갑은 삭제 불가 (is_primary = false 조건)
UPDATE wallets
SET deleted_at = NOW(), is_primary = false, updated_at = NOW()
WHERE id = ? AND user_id = ? AND is_primary = false AND deleted_at IS NULL;

-- ============================================================================
-- 지갑 존재 여부 체크
-- ============================================================================

-- name: ExistsWalletByAddress :one
-- 지갑 주소 중복 체크 (address는 lower-case로 전달, 삭제 제외)
SELECT EXISTS(
    SELECT 1 FROM wallets WHERE address = ? AND deleted_at IS NULL
) as exists_flag;

-- name: ExistsVerifiedWalletByUser :one
-- 사용자의 검증된 지갑 존재 여부 (삭제 제외)
SELECT EXISTS(
    SELECT 1 FROM wallets WHERE user_id = ? AND is_verified = true AND deleted_at IS NULL
) as exists_flag;
