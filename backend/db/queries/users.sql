-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, created_at, deleted_at, deleted_by
FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING id, email, password_hash, role, created_at, deleted_at, deleted_by;
