-- Wallet Soft Delete 롤백

ALTER TABLE wallets DROP INDEX uk_wallet_address_active;
ALTER TABLE wallets DROP COLUMN address_active;
ALTER TABLE wallets ADD UNIQUE KEY uk_wallet_address (address);
DROP INDEX idx_wallets_user_not_deleted ON wallets;
ALTER TABLE wallets DROP COLUMN deleted_at;
