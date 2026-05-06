-- name: CreateMessage :exec
INSERT INTO messages (
    msg_id,
    client_msg_id,
    sender_id,
    room_id,
    msg_type,
    server_time,
    reply_to_msg_id,
    payload,
    ext
) VALUES (
    sqlc.arg(msg_id),
    sqlc.arg(client_msg_id),
    sqlc.arg(sender_id),
    sqlc.arg(room_id),
    sqlc.arg(msg_type),
    sqlc.arg(server_time),
    sqlc.arg(reply_to_msg_id),
    sqlc.arg(payload),
    sqlc.arg(ext)
);

-- name: BatchCreateMessages :batchexec
INSERT INTO messages (
    msg_id,
    client_msg_id,
    sender_id,
    room_id,
    msg_type,
    server_time,
    reply_to_msg_id,
    payload,
    ext
) VALUES (
    sqlc.arg(msg_id),
    sqlc.arg(client_msg_id),
    sqlc.arg(sender_id),
    sqlc.arg(room_id),
    sqlc.arg(msg_type),
    sqlc.arg(server_time),
    sqlc.arg(reply_to_msg_id),
    sqlc.arg(payload),
    sqlc.arg(ext)
)
ON CONFLICT (msg_id) DO NOTHING;


-- name: BatchCopyMessages :copyfrom
INSERT INTO messages (
    msg_id,
    client_msg_id,
    sender_id,
    room_id,
    msg_type,
    server_time,
    reply_to_msg_id,
    payload,
    ext
) VALUES (
    sqlc.arg(msg_id),
    sqlc.arg(client_msg_id),
    sqlc.arg(sender_id),
    sqlc.arg(room_id),
    sqlc.arg(msg_type),
    sqlc.arg(server_time),
    sqlc.arg(reply_to_msg_id),
    sqlc.arg(payload),
    sqlc.arg(ext)
);

-- name: GetMessageByID :one
SELECT
    msg_id,
    client_msg_id,
    sender_id,
    room_id,
    msg_type,
    server_time,
    reply_to_msg_id,
    payload,
    ext
FROM messages
WHERE msg_id = sqlc.arg(msg_id)
LIMIT 1;

-- name: ListMessagesByRoom :many
SELECT
    msg_id,
    client_msg_id,
    sender_id,
    room_id,
    msg_type,
    server_time,
    reply_to_msg_id,
    payload,
    ext
FROM messages
WHERE room_id = sqlc.arg(room_id)
  AND server_time < sqlc.arg(before_server_time)
ORDER BY server_time DESC
LIMIT sqlc.arg(page_size);

-- name: GetRoomMembers :many
SELECT user_id FROM room_members WHERE room_id = sqlc.arg(room_id);

-- name: GetMembersByRoomIDs :many
SELECT room_id, user_id FROM room_members WHERE room_id = ANY(sqlc.arg(room_ids)::uuid[]);

-- name: GetUserRooms :many
SELECT room_id FROM room_members WHERE user_id = sqlc.arg(user_id);

-- name: GetRoomByHash :one
SELECT room_id, chat_type, name, avatar_url, single_chat_hash, created_at, updated_at
FROM rooms
WHERE single_chat_hash = sqlc.arg(hash) AND chat_type = 'single'
LIMIT 1;

-- name: CreateRoom :one
INSERT INTO rooms (room_id, chat_type, name, avatar_url, single_chat_hash)
VALUES (sqlc.arg(room_id), sqlc.arg(chat_type), sqlc.arg(name), sqlc.arg(avatar_url), sqlc.arg(single_chat_hash))
RETURNING room_id;

-- name: AddRoomMember :exec
INSERT INTO room_members (room_id, user_id, role)
VALUES (sqlc.arg(room_id), sqlc.arg(user_id), sqlc.arg(role))
ON CONFLICT (room_id, user_id) DO NOTHING;

-- name: AddRoomMembers :exec
INSERT INTO room_members (room_id, user_id, role)
SELECT sqlc.arg(room_id), u.user_id, 'member'
FROM UNNEST(sqlc.arg(user_ids)::uuid[]) AS u(user_id)
ON CONFLICT (room_id, user_id) DO NOTHING;

-- name: CreateGroupRoom :one
INSERT INTO rooms (room_id, chat_type, name, avatar_url)
VALUES (sqlc.arg(room_id), 'group', sqlc.arg(name), sqlc.arg(avatar_url))
RETURNING room_id;
