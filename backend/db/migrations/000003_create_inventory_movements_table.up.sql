CREATE TABLE inventory_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    type TEXT NOT NULL CHECK (type IN ('PURCHASE', 'SALE', 'ADJUSTMENT')),
    quantity INTEGER NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id)
);

CREATE INDEX idx_inventory_movements_product_id ON inventory_movements(product_id);
CREATE INDEX idx_inventory_movements_created_by ON inventory_movements(created_by);
CREATE INDEX idx_inventory_movements_deleted_by ON inventory_movements(deleted_by);
CREATE INDEX idx_inventory_movements_deleted_at ON inventory_movements(deleted_at);
CREATE INDEX idx_inventory_movements_created_at ON inventory_movements(created_at);
