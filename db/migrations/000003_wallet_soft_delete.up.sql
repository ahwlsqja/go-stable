-- Wallet Soft Delete 지원
-- deleted_at 컬럼 추가 (이미 존재하면 무시)

-- ALTER TABLE wallets
-- ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;

-- 삭제된 지갑 제외 인덱스 (조회 성능) - 이미 존재하면 무시
-- CREATE INDEX idx_wallets_user_not_deleted ON wallets(user_id, deleted_at);

-- 주소 unique 제약: 삭제되지 않은 지갑만
-- 기존 uk_address 삭제 후 partial unique index로 변경
ALTER TABLE wallets DROP INDEX uk_address;

-- MySQL은 partial index를 지원하지 않으므로,
-- deleted_at을 포함한 unique index + application level 검증으로 처리
-- 또는 generated column 사용
ALTER TABLE wallets
ADD COLUMN address_active VARCHAR(42) GENERATED ALWAYS AS (
    CASE WHEN deleted_at IS NULL THEN address ELSE NULL END
) STORED;

ALTER TABLE wallets
ADD UNIQUE KEY uk_wallet_address_active (address_active);
