-- name: CreateDeposit :execresult
INSERT INTO deposits (idempotency_key, account_id, amount, status)
VALUES (?, ?, ?, 'PENDING_MINT');

-- name: GetDeposit :one
SELECT * FROM deposits WHERE id = ? LIMIT 1;

-- name: GetDepositByIdempotencyKey :one
SELECT * FROM deposits WHERE idempotency_key = ? LIMIT 1;

-- name: UpdateDepositStatus :exec
UPDATE deposits
SET status = ?, tx_hash = ?, confirmations = ?, error_message = ?, retry_count = ?
WHERE id = ?;

-- name: ListDepositsByStatus :many
SELECT * FROM deposits WHERE status = ? ORDER BY created_at LIMIT ?;

-- name: IncrementDepositRetry :exec
UPDATE deposits SET retry_count = retry_count + 1 WHERE id = ?;
