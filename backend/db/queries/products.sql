-- name: CreateProduct :one
INSERT INTO products (name, category, sale_price_cents, cost_cents, current_stock, min_stock, image_url, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetProductByID :one
SELECT * FROM products
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTrashedProductByID :one
SELECT * FROM products
WHERE id = $1 AND deleted_at IS NOT NULL;

-- name: ListProducts :many
SELECT * FROM products
WHERE deleted_at IS NULL
  AND (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateProduct :one
UPDATE products
SET name = $2,
    category = $3,
    sale_price_cents = $4,
    cost_cents = $5,
    current_stock = $6,
    min_stock = $7,
    image_url = $8,
    version = version + 1,
    updated_at = now(),
    updated_by = $9
WHERE id = $1 AND version = $10 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProduct :execrows
UPDATE products
SET deleted_at = now(), deleted_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: RestoreProduct :one
UPDATE products
SET deleted_at = NULL, deleted_by = NULL
WHERE id = $1 AND deleted_at IS NOT NULL
RETURNING *;

-- name: ListTrashedProducts :many
SELECT p.*, u.email AS deleted_by_email
FROM products p
LEFT JOIN users u ON u.id = p.deleted_by
WHERE p.deleted_at IS NOT NULL
ORDER BY p.deleted_at DESC;

-- name: ListLowStockProducts :many
SELECT * FROM products
WHERE deleted_at IS NULL AND current_stock <= min_stock
ORDER BY current_stock ASC;
