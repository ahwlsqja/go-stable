-- name: GetAccount :one
SELECT * FROM accounts WHERE id = ? LIMIT 1;

-- name: GetAccountByExternalID :one
SELECT * FROM accounts WHERE external_id = ? LIMIT 1;

-- name: CreateAccount :execresult
INSERT INTO accounts (external_id, account_type, status)
VALUES (?, ?, ?);

-- name: UpdateAccountStatus :exec
UPDATE accounts SET status = ? WHERE id = ?;

-- name: GetBalance :one
SELECT * FROM balances WHERE account_id = ? AND currency = ? LIMIT 1;

-- name: GetBalanceForUpdate :one
SELECT * FROM balances WHERE account_id = ? AND currency = ? FOR UPDATE;

-- name: CreateBalance :execresult
INSERT INTO balances (account_id, currency, available, held, version)
VALUES (?, ?, ?, ?, 0);

-- name: UpdateBalanceOptimistic :execresult
UPDATE balances
SET available = ?, held = ?, version = version + 1
WHERE id = ? AND version = ?;

-- name: ListAccountsByType :many
SELECT * FROM accounts WHERE account_type = ? AND status = 'ACTIVE';
