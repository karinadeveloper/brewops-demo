-- name: GetRevenueTotal :one
SELECT COALESCE(SUM(total_cents), 0)::bigint AS total_cents
FROM sales
WHERE deleted_at IS NULL
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date'));

-- name: GetRevenueByDay :many
-- "day" is the calendar day in America/Mexico_City, NOT in UTC (the
-- session's own timezone) — created_at is converted to Mexico City
-- wall-clock time before truncating to midnight, then converted back to
-- the UTC instant that midnight corresponds to. Without this, a sale made
-- at, say, 11pm in Mexico City (already past midnight in UTC) would be
-- miscounted into the next calendar day from the business owner's
-- perspective. The returned value is still a timestamptz — an absolute
-- instant — but it always lands exactly on a Mexico City midnight.
SELECT (date_trunc('day', created_at AT TIME ZONE 'America/Mexico_City') AT TIME ZONE 'America/Mexico_City')::timestamptz AS day,
       COALESCE(SUM(total_cents), 0)::bigint AS total_cents
FROM sales
WHERE deleted_at IS NULL
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date'))
GROUP BY day
ORDER BY day ASC;

-- name: ListTopProductsByRevenue :many
SELECT p.id, p.name,
       COALESCE(SUM(si.quantity), 0)::int AS quantity_sold,
       COALESCE(SUM(si.quantity * si.unit_price_cents), 0)::bigint AS revenue_cents
FROM sale_items si
JOIN sales s ON s.id = si.sale_id
JOIN products p ON p.id = si.product_id
WHERE s.deleted_at IS NULL
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR s.created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR s.created_at <= sqlc.narg('to_date'))
GROUP BY p.id, p.name
ORDER BY revenue_cents DESC
LIMIT sqlc.arg('limit');

-- name: ListTopProductsByQuantity :many
SELECT p.id, p.name,
       COALESCE(SUM(si.quantity), 0)::int AS quantity_sold,
       COALESCE(SUM(si.quantity * si.unit_price_cents), 0)::bigint AS revenue_cents
FROM sale_items si
JOIN sales s ON s.id = si.sale_id
JOIN products p ON p.id = si.product_id
WHERE s.deleted_at IS NULL
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR s.created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR s.created_at <= sqlc.narg('to_date'))
GROUP BY p.id, p.name
ORDER BY quantity_sold DESC
LIMIT sqlc.arg('limit');

-- name: GetInventoryValue :one
SELECT COALESCE(SUM(current_stock::bigint * cost_cents), 0)::bigint AS total_value_cents
FROM products
WHERE deleted_at IS NULL;
