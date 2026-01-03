-- name: UpsertAllowlistEntry :exec
INSERT INTO rate_limit_allowlist (id, entry_type, identifier, reason, expires_at, created_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (entry_type, identifier) DO UPDATE SET
    reason = EXCLUDED.reason,
    expires_at = EXCLUDED.expires_at;

-- name: DeleteAllowlistEntry :exec
DELETE FROM rate_limit_allowlist WHERE entry_type = $1 AND identifier = $2;

-- name: IsAllowlisted :one
SELECT EXISTS(
    SELECT 1
    FROM rate_limit_allowlist
    WHERE identifier = $1
      AND (expires_at IS NULL OR expires_at > $2)
);

-- name: ListAllowlistEntries :many
SELECT id, entry_type, identifier, reason, expires_at, created_at, created_by
FROM rate_limit_allowlist
WHERE expires_at IS NULL OR expires_at > $1;

-- name: DeleteExpiredAllowlistEntries :exec
DELETE FROM rate_limit_allowlist WHERE expires_at IS NOT NULL AND expires_at <= $1;
