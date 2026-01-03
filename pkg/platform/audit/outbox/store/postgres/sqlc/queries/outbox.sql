-- name: InsertOutboxEntry :exec
INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListUnprocessedOutboxEntries :many
SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, processed_at
FROM outbox
WHERE processed_at IS NULL
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEntryProcessed :execresult
UPDATE outbox
SET processed_at = $2
WHERE id = $1 AND processed_at IS NULL;

-- name: CountPendingOutboxEntries :one
SELECT COUNT(*) FROM outbox WHERE processed_at IS NULL;

-- name: DeleteProcessedOutboxEntriesBefore :execresult
DELETE FROM outbox WHERE processed_at IS NOT NULL AND processed_at < $1;
