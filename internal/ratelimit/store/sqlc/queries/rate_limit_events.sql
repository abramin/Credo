-- name: LockRateLimitKey :exec
SELECT pg_advisory_xact_lock(hashtext($1)::bigint);

-- name: DeleteRateLimitEventsBefore :exec
DELETE FROM rate_limit_events WHERE key = $1 AND occurred_at < $2;

-- name: SumRateLimitCost :one
SELECT COALESCE(SUM(cost), 0)::bigint FROM rate_limit_events WHERE key = $1;

-- name: MinRateLimitOccurredAt :one
SELECT MIN(occurred_at) AS occurred_at FROM rate_limit_events WHERE key = $1;

-- name: InsertRateLimitEvent :exec
INSERT INTO rate_limit_events (key, occurred_at, cost, window_seconds)
VALUES ($1, $2, $3, $4);

-- name: DeleteRateLimitEventsByKey :exec
DELETE FROM rate_limit_events WHERE key = $1;

-- name: GetLatestRateLimitWindowSeconds :one
SELECT window_seconds
FROM rate_limit_events
WHERE key = $1
ORDER BY occurred_at DESC
LIMIT 1;

-- name: SumRateLimitCostSince :one
SELECT COALESCE(SUM(cost), 0)::bigint
FROM rate_limit_events
WHERE key = $1 AND occurred_at >= $2;
