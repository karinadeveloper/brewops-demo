ALTER TABLE sales ADD COLUMN idempotency_key UUID NULL;
CREATE UNIQUE INDEX idx_sales_idempotency_key ON sales (idempotency_key)
  WHERE idempotency_key IS NOT NULL;
