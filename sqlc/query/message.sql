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
