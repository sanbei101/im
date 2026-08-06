-- name: CreateUser :one
INSERT INTO users (username, password)
VALUES (sqlc.arg(username), sqlc.arg(password))
RETURNING user_id, username, display_name, avatar_url, bio, created_at, updated_at;

-- name: GetUserByUsername :one
SELECT user_id, username, password, display_name, avatar_url, bio, created_at, updated_at
FROM users
WHERE username = sqlc.arg(username)
LIMIT 1;

-- name: GetUserByID :one
SELECT user_id, username, password, display_name, avatar_url, bio, created_at, updated_at
FROM users
WHERE user_id = sqlc.arg(user_id)
LIMIT 1;

-- name: UpdateUserProfile :one
UPDATE users
SET display_name = sqlc.arg(display_name),
    avatar_url = sqlc.arg(avatar_url),
    bio = sqlc.arg(bio),
    updated_at = NOW()
WHERE user_id = sqlc.arg(user_id)
RETURNING user_id, username, display_name, avatar_url, bio, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET password = sqlc.arg(password), updated_at = NOW()
WHERE user_id = sqlc.arg(user_id);

-- name: CreateRefreshSession :one
INSERT INTO refresh_sessions (user_id, token_hash, expires_at)
VALUES (sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(expires_at))
RETURNING session_id, user_id, token_hash, expires_at, revoked_at, created_at;

-- name: GetRefreshSessionForUpdate :one
SELECT session_id, user_id, token_hash, expires_at, revoked_at, created_at
FROM refresh_sessions
WHERE token_hash = sqlc.arg(token_hash)
FOR UPDATE;

-- name: RevokeRefreshSession :exec
UPDATE refresh_sessions
SET revoked_at = NOW()
WHERE session_id = sqlc.arg(session_id) AND revoked_at IS NULL;

-- name: RevokeUserRefreshSessions :exec
UPDATE refresh_sessions
SET revoked_at = NOW()
WHERE user_id = sqlc.arg(user_id) AND revoked_at IS NULL;

-- name: SearchPublicUsers :many
SELECT user_id, username, display_name, avatar_url, bio
FROM users
WHERE username ILIKE '%' || sqlc.arg(query)::text || '%'
   OR display_name ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY username
LIMIT 20;
