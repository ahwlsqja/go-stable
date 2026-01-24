-- ============================================================================
-- Wallet Queries - Phase 1
-- ============================================================================
-- NOTE: wallet address는 저장/조회 시 lower-case normalize 적용
--       서비스 레이어에서 strings.ToLower() 처리 후 쿼리 호출

-- name: CreateWallet :execresult
-- 지갑 등록 (address는 서비스에서 lower-case 변환 후 전달)
-- is_verified=false, is_primary=false 기본값
INSERT INTO wallets (user_id, address, label, is_primary, is_verified)
VALUES (?, ?, ?, false, false);

-- name: GetWalletByID :one
-- ID로 지갑 조회
SELECT * FROM wallets WHERE id = ?;

-- name: GetWalletByIDAndUser :one
-- ID + 사용자 소유권 검증 조회
SELECT * FROM wallets
WHERE id = ? AND user_id = ?;

-- name: GetWalletByAddress :one
-- 주소로 지갑 조회 (address는 lower-case로 전달)
SELECT * FROM wallets WHERE address = ?;

-- name: GetWalletForUpdate :one
-- 트랜잭션 내 row-lock (검증, Primary 설정 등)
SELECT * FROM wallets
WHERE id = ? AND user_id = ?
FOR UPDATE;

-- name: GetPrimaryWallet :one
-- 사용자의 Primary 지갑 조회
SELECT * FROM wallets
WHERE user_id = ? AND is_primary = true
LIMIT 1;

-- name: ListWalletsByUser :many
-- 사용자의 전체 지갑 목록
SELECT * FROM wallets
WHERE user_id = ?
ORDER BY is_primary DESC, created_at ASC;

-- name: CountWalletsByUser :one
-- 사용자의 지갑 수 조회
SELECT COUNT(*) as total FROM wallets
WHERE user_id = ?;

-- ============================================================================
-- 지갑 상태 업데이트
-- ============================================================================

-- name: UpdateWalletVerified :exec
-- EIP-712 서명 검증 완료
UPDATE wallets
SET is_verified = true, updated_at = NOW()
WHERE id = ? AND user_id = ?;

-- name: UpdateWalletLabel :exec
-- 지갑 라벨 변경
UPDATE wallets
SET label = ?, updated_at = NOW()
WHERE id = ? AND user_id = ?;

-- ============================================================================
-- Primary 지갑 설정 (트랜잭션 내 호출)
-- ============================================================================

-- name: ClearPrimaryWallet :exec
-- 기존 Primary 지갑 해제 (SetPrimary 트랜잭션 첫 단계)
UPDATE wallets
SET is_primary = false, updated_at = NOW()
WHERE user_id = ? AND is_primary = true;

-- name: SetWalletPrimary :execresult
-- 새 Primary 지갑 설정 (소유권 검증 포함)
-- SetPrimary 트랜잭션: 1) GetUserForUpdate 2) ClearPrimaryWallet 3) SetWalletPrimary
UPDATE wallets
SET is_primary = true, updated_at = NOW()
WHERE id = ? AND user_id = ? AND is_verified = true;

-- ============================================================================
-- 지갑 삭제
-- ============================================================================

-- name: HardDeleteWallet :execresult
-- TODO: Phase 6 이후 소프트 삭제(deleted_at 컬럼) 전환 검토
-- Primary 지갑은 삭제 불가 (is_primary = false 조건)
DELETE FROM wallets
WHERE id = ? AND user_id = ? AND is_primary = false;

-- ============================================================================
-- 지갑 존재 여부 체크
-- ============================================================================

-- name: ExistsWalletByAddress :one
-- 지갑 주소 중복 체크 (address는 lower-case로 전달)
SELECT EXISTS(
    SELECT 1 FROM wallets WHERE address = ?
) as exists_flag;

-- name: ExistsVerifiedWalletByUser :one
-- 사용자의 검증된 지갑 존재 여부
SELECT EXISTS(
    SELECT 1 FROM wallets WHERE user_id = ? AND is_verified = true
) as exists_flag;
