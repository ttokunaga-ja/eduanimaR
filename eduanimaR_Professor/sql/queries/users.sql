-- sql/queries/users.sql

-- name: GetUserByID :one
SELECT *
FROM users
WHERE user_id = $1;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (user_id, email, provider, provider_user_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET
    email            = COALESCE(sqlc.narg(email), email),
    provider         = COALESCE(sqlc.narg(provider), provider),
    provider_user_id = COALESCE(sqlc.narg(provider_user_id), provider_user_id),
    updated_at       = NOW()
WHERE user_id = sqlc.arg(user_id)
RETURNING *;
