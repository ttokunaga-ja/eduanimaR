-- sql/queries/users.sql

-- name: GetUserByID :one
SELECT
  id AS user_id,
  provider,
  provider_user_id,
  created_at,
  updated_at
FROM users
WHERE id = $1
  AND is_active = TRUE;

-- name: GetUserByEmail :one
SELECT
  id AS user_id,
  provider,
  provider_user_id,
  created_at,
  updated_at
FROM users
WHERE provider_user_id = $1
  AND is_active = TRUE;

-- name: CreateUser :one
INSERT INTO users (id, provider, provider_user_id)
VALUES ($1, COALESCE($3, 'development'), COALESCE($4, $2))
RETURNING
  id AS user_id,
  provider,
  provider_user_id,
  created_at,
  updated_at;

-- name: UpdateUser :one
UPDATE users
SET
  provider = COALESCE(sqlc.narg(provider), provider),
  provider_user_id = COALESCE(sqlc.narg(provider_user_id), provider_user_id),
  updated_at = NOW()
WHERE id = sqlc.arg(user_id)
RETURNING
  id AS user_id,
  provider,
  provider_user_id,
  created_at,
  updated_at;
