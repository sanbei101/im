-- name: CreateUser :one
INSERT INTO users (username, password)
VALUES (sqlc.arg(username), sqlc.arg(password))
RETURNING user_id, username, created_at;

-- name: GetUserByUsername :one
SELECT user_id, username, password, created_at
FROM users
WHERE username = sqlc.arg(username)
LIMIT 1;

-- name: BatchCreateUsers :batchone
INSERT INTO users (username, password)
VALUES (sqlc.arg(username), sqlc.arg(password))
RETURNING user_id;
