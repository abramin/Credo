-- name: InsertOutboxEntry :exec
INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertAuditEvent :exec
INSERT INTO audit_events (
    id, category, timestamp, user_id, subject, action,
    purpose, requesting_party, decision, reason,
    email, request_id, actor_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO NOTHING;

-- name: ListAuditEventsByUser :many
SELECT category, timestamp, user_id, subject, action,
       purpose, requesting_party, decision, reason,
       email, request_id, actor_id
FROM audit_events
WHERE user_id = $1
ORDER BY timestamp DESC;

-- name: ListAuditEvents :many
SELECT category, timestamp, user_id, subject, action,
       purpose, requesting_party, decision, reason,
       email, request_id, actor_id
FROM audit_events
ORDER BY timestamp DESC;

-- name: ListRecentAuditEvents :many
SELECT category, timestamp, user_id, subject, action,
       purpose, requesting_party, decision, reason,
       email, request_id, actor_id
FROM audit_events
ORDER BY timestamp DESC
LIMIT $1;
