CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'ADMIN' CHECK (role = 'ADMIN'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id)
);

CREATE INDEX idx_users_deleted_by ON users(deleted_by);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
