-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, created_at, deleted_at, deleted_by
FROM users
WHERE email = $1 AND deleted_at IS NULL;
