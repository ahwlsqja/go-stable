-- name: CreateOutboxJob :execresult
INSERT INTO outbox_jobs (job_type, reference_id, payload, status, max_retries, next_retry_at)
VALUES (?, ?, ?, 'PENDING', ?, NOW());

-- name: GetOutboxJob :one
SELECT * FROM outbox_jobs WHERE id = ? LIMIT 1;

-- name: GetPendingOutboxJobs :many
SELECT * FROM outbox_jobs
WHERE status IN ('PENDING', 'FAILED')
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
  AND (locked_until IS NULL OR locked_until <= NOW())
  AND retry_count < max_retries
ORDER BY created_at
LIMIT ?;

-- name: LockOutboxJob :execresult
UPDATE outbox_jobs
SET status = 'PROCESSING', locked_until = ?
WHERE id = ? AND (locked_until IS NULL OR locked_until <= NOW());

-- name: CompleteOutboxJob :exec
UPDATE outbox_jobs
SET status = 'COMPLETED', locked_until = NULL
WHERE id = ?;

-- name: FailOutboxJob :exec
UPDATE outbox_jobs
SET status = CASE WHEN retry_count >= max_retries - 1 THEN 'DEAD_LETTER' ELSE 'FAILED' END,
    retry_count = retry_count + 1,
    next_retry_at = ?,
    error_message = ?,
    locked_until = NULL
WHERE id = ?;

-- name: GetOutboxJobByTypeAndRef :one
SELECT * FROM outbox_jobs WHERE job_type = ? AND reference_id = ? LIMIT 1;
