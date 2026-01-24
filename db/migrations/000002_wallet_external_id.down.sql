-- ============================================================================
-- Wallet external_id 제거
-- ============================================================================

ALTER TABLE wallets
DROP KEY uk_wallet_external_id,
DROP COLUMN external_id;
