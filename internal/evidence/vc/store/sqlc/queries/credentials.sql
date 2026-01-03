-- name: UpsertCredential :exec
INSERT INTO vc_credentials (id, type, subject_id, issuer, issued_at, claims, is_over_18, verified_via)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE SET
    type = EXCLUDED.type,
    subject_id = EXCLUDED.subject_id,
    issuer = EXCLUDED.issuer,
    issued_at = EXCLUDED.issued_at,
    claims = EXCLUDED.claims,
    is_over_18 = EXCLUDED.is_over_18,
    verified_via = EXCLUDED.verified_via;

-- name: GetCredentialByID :one
SELECT id, type, subject_id, issuer, issued_at,
    CASE WHEN type = 'AgeOver18' THEN NULL::jsonb ELSE claims END AS claims,
    is_over_18, verified_via
FROM vc_credentials
WHERE id = $1;

-- name: GetCredentialBySubjectAndType :one
SELECT id, type, subject_id, issuer, issued_at,
    CASE WHEN type = 'AgeOver18' THEN NULL::jsonb ELSE claims END AS claims,
    is_over_18, verified_via
FROM vc_credentials
WHERE subject_id = $1 AND type = $2
ORDER BY issued_at DESC
LIMIT 1;
