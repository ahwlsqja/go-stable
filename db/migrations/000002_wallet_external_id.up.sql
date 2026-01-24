-- ============================================================================
-- Wallet external_id 추가
-- ============================================================================

ALTER TABLE wallets
ADD COLUMN external_id VARCHAR(64) AFTER id;

-- 기존 데이터가 있다면 UUID 생성 (신규 프로젝트라 불필요하지만 안전을 위해)
-- UPDATE wallets SET external_id = UUID() WHERE external_id IS NULL;

-- NOT NULL + UNIQUE 제약 추가
ALTER TABLE wallets
MODIFY COLUMN external_id VARCHAR(64) NOT NULL,
ADD UNIQUE KEY uk_wallet_external_id (external_id);
