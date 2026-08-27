CREATE TABLE sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
    payment_method TEXT NOT NULL CHECK (payment_method IN ('CASH', 'TRANSFER')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id)
);

CREATE TABLE sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales(id),
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents BIGINT NOT NULL CHECK (unit_price_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sales_created_by ON sales(created_by);
CREATE INDEX idx_sales_deleted_by ON sales(deleted_by);
CREATE INDEX idx_sales_deleted_at ON sales(deleted_at);
CREATE INDEX idx_sales_created_at ON sales(created_at);
CREATE INDEX idx_sale_items_sale_id ON sale_items(sale_id);
CREATE INDEX idx_sale_items_product_id ON sale_items(product_id);
