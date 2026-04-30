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
