-- name: CreateLedgerEntry :execresult
INSERT INTO ledger_entries (tx_id, account_id, entry_type, amount, balance_after, description, idempotency_key)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetLedgerEntriesByTxID :many
SELECT * FROM ledger_entries WHERE tx_id = ? ORDER BY id;

-- name: GetLedgerEntriesByAccount :many
SELECT * FROM ledger_entries
WHERE account_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: GetLedgerEntryByIdempotencyKey :one
SELECT * FROM ledger_entries WHERE idempotency_key = ? LIMIT 1;

-- name: SumLedgerEntriesByType :one
SELECT
    COALESCE(SUM(CASE WHEN entry_type = 'CREDIT' THEN amount ELSE 0 END), 0) as total_credits,
    COALESCE(SUM(CASE WHEN entry_type = 'DEBIT' THEN amount ELSE 0 END), 0) as total_debits
FROM ledger_entries
WHERE account_id = ?;
