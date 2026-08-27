CREATE TABLE marketing_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    image_url TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('PRODUCT', 'PROMOTION')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id)
);

CREATE INDEX idx_marketing_assets_created_by ON marketing_assets(created_by);
CREATE INDEX idx_marketing_assets_deleted_by ON marketing_assets(deleted_by);
CREATE INDEX idx_marketing_assets_deleted_at ON marketing_assets(deleted_at);
