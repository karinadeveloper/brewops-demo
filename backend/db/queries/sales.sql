-- name: CreateSale :one
INSERT INTO sales (total_cents, payment_method, created_by)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateSaleItem :one
INSERT INTO sale_items (sale_id, product_id, quantity, unit_price_cents)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DecrementProductStockForSale :one
-- Mirrors UpdateProduct's optimistic-concurrency guard: the WHERE version
-- match means a concurrent sale (or edit) of the same product between the
-- caller's stock check and this decrement makes this return zero rows,
-- which the repository surfaces as ErrOptimisticLockConflict (409).
UPDATE products
SET current_stock = current_stock - sqlc.arg('quantity')::integer,
    version = version + 1,
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND version = sqlc.arg('version')
  AND deleted_at IS NULL
RETURNING *;

-- name: GetSaleByID :one
SELECT * FROM sales
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTrashedSaleByID :one
SELECT * FROM sales
WHERE id = $1 AND deleted_at IS NOT NULL;

-- name: ListSaleItemsBySaleID :many
SELECT * FROM sale_items
WHERE sale_id = $1
ORDER BY created_at ASC;

-- name: ListSales :many
SELECT * FROM sales
WHERE deleted_at IS NULL
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: SoftDeleteSale :execrows
UPDATE sales
SET deleted_at = now(), deleted_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: RestoreSale :one
UPDATE sales
SET deleted_at = NULL, deleted_by = NULL
WHERE id = $1 AND deleted_at IS NOT NULL
RETURNING *;

-- name: ListTrashedSales :many
SELECT s.*, u.email AS deleted_by_email
FROM sales s
LEFT JOIN users u ON u.id = s.deleted_by
WHERE s.deleted_at IS NOT NULL
ORDER BY s.deleted_at DESC;
