-- name: GetAuthLockout :one
SELECT identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha
FROM auth_lockouts
WHERE identifier = $1;

-- name: GetOrCreateAuthLockout :one
INSERT INTO auth_lockouts (identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha)
VALUES ($1, 0, 0, NULL, $2, FALSE)
ON CONFLICT (identifier) DO UPDATE SET
    identifier = EXCLUDED.identifier
RETURNING identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha;

-- name: DeleteAuthLockout :exec
DELETE FROM auth_lockouts WHERE identifier = $1;

-- name: UpsertAuthLockout :exec
INSERT INTO auth_lockouts (identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (identifier) DO UPDATE SET
    failure_count = EXCLUDED.failure_count,
    daily_failures = EXCLUDED.daily_failures,
    locked_until = EXCLUDED.locked_until,
    last_failure_at = EXCLUDED.last_failure_at,
    requires_captcha = EXCLUDED.requires_captcha;

-- name: RecordFailureAtomic :one
INSERT INTO auth_lockouts (identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha)
VALUES ($1, 1, 1, NULL, $2, FALSE)
ON CONFLICT (identifier) DO UPDATE SET
    failure_count = auth_lockouts.failure_count + 1,
    daily_failures = auth_lockouts.daily_failures + 1,
    last_failure_at = $2
RETURNING identifier, failure_count, daily_failures, locked_until, last_failure_at, requires_captcha;

-- name: ApplyHardLock :execresult
UPDATE auth_lockouts
SET locked_until = $2
WHERE identifier = $1
  AND daily_failures >= $3
  AND (locked_until IS NULL OR locked_until < NOW());

-- name: SetRequiresCaptcha :execresult
UPDATE auth_lockouts
SET requires_captcha = TRUE
WHERE identifier = $1
  AND requires_captcha = FALSE
  AND daily_failures >= $2;

-- name: SumFailureCountBefore :one
SELECT COALESCE(SUM(failure_count), 0)::bigint
FROM auth_lockouts
WHERE last_failure_at < $1;

-- name: ResetFailureCountBefore :exec
UPDATE auth_lockouts SET failure_count = 0 WHERE last_failure_at < $1;

-- name: SumDailyFailuresBefore :one
SELECT COALESCE(SUM(daily_failures), 0)::bigint
FROM auth_lockouts
WHERE last_failure_at < $1;

-- name: ResetDailyFailuresBefore :exec
UPDATE auth_lockouts SET daily_failures = 0 WHERE last_failure_at < $1;
