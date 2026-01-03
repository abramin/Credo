-- name: GetCitizenCache :one
SELECT national_id, full_name, date_of_birth, address, valid, source, checked_at, regulated
FROM citizen_cache
WHERE national_id = $1 AND regulated = $2 AND checked_at >= $3;

-- name: UpsertCitizenCache :exec
INSERT INTO citizen_cache (
    national_id, full_name, date_of_birth, address, valid, source, checked_at, regulated
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (national_id, regulated) DO UPDATE SET
    full_name = EXCLUDED.full_name,
    date_of_birth = EXCLUDED.date_of_birth,
    address = EXCLUDED.address,
    valid = EXCLUDED.valid,
    source = EXCLUDED.source,
    checked_at = EXCLUDED.checked_at;

-- name: GetSanctionsCache :one
SELECT national_id, listed, source, checked_at
FROM sanctions_cache
WHERE national_id = $1 AND checked_at >= $2;

-- name: UpsertSanctionsCache :exec
INSERT INTO sanctions_cache (national_id, listed, source, checked_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (national_id) DO UPDATE SET
    listed = EXCLUDED.listed,
    source = EXCLUDED.source,
    checked_at = EXCLUDED.checked_at;
