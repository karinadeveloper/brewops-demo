-- name: CreateMarketingAsset :one
INSERT INTO marketing_assets (name, image_url, type, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListMarketingAssets :many
SELECT * FROM marketing_assets
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: SoftDeleteMarketingAsset :execrows
UPDATE marketing_assets
SET deleted_at = now(), deleted_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: RestoreMarketingAsset :one
UPDATE marketing_assets
SET deleted_at = NULL, deleted_by = NULL
WHERE id = $1 AND deleted_at IS NOT NULL
RETURNING *;

-- name: ListTrashedMarketingAssets :many
SELECT a.*, u.email AS deleted_by_email
FROM marketing_assets a
LEFT JOIN users u ON u.id = a.deleted_by
WHERE a.deleted_at IS NOT NULL
ORDER BY a.deleted_at DESC;
