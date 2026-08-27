CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('juice', 'water', 'soda', 'other')),
    sale_price_cents BIGINT NOT NULL CHECK (sale_price_cents >= 0),
    cost_cents BIGINT NOT NULL CHECK (cost_cents >= 0),
    current_stock INTEGER NOT NULL DEFAULT 0 CHECK (current_stock >= 0),
    min_stock INTEGER NOT NULL DEFAULT 0 CHECK (min_stock >= 0),
    image_url TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id)
);

CREATE INDEX idx_products_updated_by ON products(updated_by);
CREATE INDEX idx_products_deleted_by ON products(deleted_by);
CREATE INDEX idx_products_deleted_at ON products(deleted_at);
CREATE INDEX idx_products_category ON products(category);
