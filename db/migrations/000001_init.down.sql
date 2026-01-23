-- ============================================================================
-- B2B Commerce Settlement Engine - Drop All Tables (Reverse Order)
-- ============================================================================

-- Layer 6: Infrastructure
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS outbox;

-- Layer 5: Payment & Settlement
DROP TABLE IF EXISTS settlements;
DROP TABLE IF EXISTS payments;

-- Layer 4: Bridge
DROP TABLE IF EXISTS withdrawals;
DROP TABLE IF EXISTS deposits;

-- Layer 3: Account & Ledger
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS accounts;

-- Layer 2: Commerce
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS inventory_logs;
DROP TABLE IF EXISTS inventories;
DROP TABLE IF EXISTS products;

-- Layer 1: Identity & Wallet
DROP TABLE IF EXISTS system_wallets;
DROP TABLE IF EXISTS wallets;
DROP TABLE IF EXISTS users;
