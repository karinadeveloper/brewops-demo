DROP INDEX IF EXISTS idx_sales_idempotency_key;
ALTER TABLE sales DROP COLUMN IF EXISTS idempotency_key;
