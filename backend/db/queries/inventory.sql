-- name: CreateInventoryMovement :one
INSERT INTO inventory_movements (product_id, type, quantity, reason, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: AdjustProductStock :one
-- Atomically applies a signed delta to current_stock, guarded so the update
-- affects zero rows (rather than violating the CHECK constraint) when the
-- product is missing/soft-deleted or the delta would take stock negative —
-- the caller distinguishes those cases with a follow-up GetProductByID.
UPDATE products
SET current_stock = current_stock + sqlc.arg('delta')::integer,
    version = version + 1,
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND deleted_at IS NULL
  AND current_stock + sqlc.arg('delta')::integer >= 0
RETURNING *;

-- name: GetInventoryMovementByID :one
SELECT * FROM inventory_movements
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTrashedInventoryMovementByID :one
SELECT * FROM inventory_movements
WHERE id = $1 AND deleted_at IS NOT NULL;

-- name: ListInventoryMovements :many
SELECT * FROM inventory_movements
WHERE deleted_at IS NULL
  AND (sqlc.narg('product_id')::uuid IS NULL OR product_id = sqlc.narg('product_id'))
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: SoftDeleteInventoryMovement :execrows
UPDATE inventory_movements
SET deleted_at = now(), deleted_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: RestoreInventoryMovement :one
UPDATE inventory_movements
SET deleted_at = NULL, deleted_by = NULL
WHERE id = $1 AND deleted_at IS NOT NULL
RETURNING *;

-- name: ListTrashedInventoryMovements :many
SELECT m.*, u.email AS deleted_by_email
FROM inventory_movements m
LEFT JOIN users u ON u.id = m.deleted_by
WHERE m.deleted_at IS NOT NULL
ORDER BY m.deleted_at DESC;
