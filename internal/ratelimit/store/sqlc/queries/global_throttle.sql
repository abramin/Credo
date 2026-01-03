-- name: GetGlobalThrottleBucket :one
SELECT bucket_start, count
FROM global_throttle
WHERE bucket_type = $1;

-- name: GetGlobalThrottleBucketForUpdate :one
SELECT bucket_start, count
FROM global_throttle
WHERE bucket_type = $1
FOR UPDATE;

-- name: InsertGlobalThrottleBucket :exec
INSERT INTO global_throttle (bucket_type, bucket_start, count)
VALUES ($1, $2, 0)
ON CONFLICT (bucket_type) DO NOTHING;

-- name: UpdateGlobalThrottleBucket :exec
UPDATE global_throttle
SET bucket_start = $2, count = $3
WHERE bucket_type = $1;
