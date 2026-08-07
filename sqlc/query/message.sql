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
    ext,
    (recalled_at IS NOT NULL)::boolean AS is_recalled
FROM messages
WHERE room_id = sqlc.arg(room_id)
  AND server_time < sqlc.arg(before_server_time)
ORDER BY server_time DESC, msg_id DESC
LIMIT sqlc.arg(page_size);

-- name: GetMembersByRoomIDs :many
SELECT room_id, user_id FROM room_members WHERE room_id = ANY(sqlc.arg(room_ids)::uuid[]);

-- name: GetUserRooms :many
SELECT r.room_id, r.chat_type, r.name, r.avatar_url, r.single_chat_hash, r.created_at, r.updated_at,
       rm.is_hidden, rm.is_muted,
       COALESCE((SELECT m.server_time FROM messages m WHERE m.room_id = r.room_id ORDER BY m.server_time DESC, m.msg_id DESC LIMIT 1), 0)::bigint AS last_message_server_time,
       (SELECT COUNT(*)::bigint FROM messages m2 WHERE m2.room_id = r.room_id AND m2.sender_id <> rm.user_id AND m2.server_time > rm.last_read_server_time AND m2.recalled_at IS NULL)::bigint AS unread_count
FROM rooms r
INNER JOIN room_members rm ON r.room_id = rm.room_id
WHERE rm.user_id = sqlc.arg(user_id)
ORDER BY last_message_server_time DESC, r.room_id;

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

-- name: IsRoomMember :one
SELECT EXISTS (
    SELECT 1
    FROM room_members
    WHERE room_id = sqlc.arg(room_id)
      AND user_id = sqlc.arg(user_id)
);

-- name: InsertMessage :one
INSERT INTO messages (
    msg_id, client_msg_id, sender_id, room_id, msg_type, server_time,
    reply_to_msg_id, payload, ext
) VALUES (
    sqlc.arg(msg_id), sqlc.arg(client_msg_id), sqlc.arg(sender_id), sqlc.arg(room_id),
    sqlc.arg(msg_type), sqlc.arg(server_time), sqlc.arg(reply_to_msg_id),
    sqlc.arg(payload), sqlc.arg(ext)
)
ON CONFLICT (sender_id, client_msg_id) DO NOTHING
RETURNING msg_id, client_msg_id, sender_id, room_id, msg_type, server_time, reply_to_msg_id, payload, ext;

-- name: GetMessageByClientID :one
SELECT msg_id, client_msg_id, sender_id, room_id, msg_type, server_time, reply_to_msg_id, payload, ext,
       (recalled_at IS NOT NULL)::boolean AS is_recalled
FROM messages
WHERE sender_id = sqlc.arg(sender_id)
  AND client_msg_id = sqlc.arg(client_msg_id)
LIMIT 1;

-- name: ListMessagesAfterTime :many
SELECT
    m.msg_id,
    m.client_msg_id,
    m.sender_id,
    m.room_id,
    m.msg_type,
    m.server_time,
    m.reply_to_msg_id,
    m.payload,
    m.ext,
    (m.recalled_at IS NOT NULL)::boolean AS is_recalled
FROM messages m
INNER JOIN room_members rm ON rm.room_id = m.room_id
WHERE rm.user_id = sqlc.arg(user_id)
  AND m.server_time > sqlc.arg(after_server_time)
ORDER BY m.server_time ASC, m.msg_id ASC
LIMIT sqlc.arg(page_size);

-- name: GetRoomByID :one
SELECT room_id, chat_type, name, avatar_url, single_chat_hash, created_at, updated_at
FROM rooms
WHERE room_id = sqlc.arg(room_id);

-- name: GetRoomMemberRole :one
SELECT role
FROM room_members
WHERE room_id = sqlc.arg(room_id) AND user_id = sqlc.arg(user_id);

-- name: ListRoomMembers :many
SELECT rm.user_id, rm.role, rm.is_hidden, rm.is_muted, rm.joined_at,
       u.username, u.display_name, u.avatar_url, u.bio
FROM room_members rm
JOIN users u ON u.user_id = rm.user_id
WHERE rm.room_id = sqlc.arg(room_id)
ORDER BY CASE rm.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, rm.user_id;

-- name: ListRoomMembersByRoomIDs :many
SELECT room_id, user_id, joined_at, is_muted
FROM room_members
WHERE room_id = ANY(sqlc.arg(room_ids)::uuid[]);

-- name: DeleteRoomMember :exec
DELETE FROM room_members
WHERE room_id = sqlc.arg(room_id) AND user_id = sqlc.arg(user_id);

-- name: DeleteRoom :exec
DELETE FROM rooms WHERE room_id = sqlc.arg(room_id);

-- name: UpdateRoomMemberRole :exec
UPDATE room_members
SET role = sqlc.arg(role)
WHERE room_id = sqlc.arg(room_id) AND user_id = sqlc.arg(user_id);

-- name: UpdateRoomMemberSettings :exec
UPDATE room_members
SET is_hidden = sqlc.arg(is_hidden), is_muted = sqlc.arg(is_muted)
WHERE room_id = sqlc.arg(room_id) AND user_id = sqlc.arg(user_id);

-- name: GetRoomMemberSettings :one
SELECT is_hidden, is_muted
FROM room_members
WHERE room_id = sqlc.arg(room_id) AND user_id = sqlc.arg(user_id);

-- name: UpdateRoomReadCursor :exec
UPDATE room_members
SET last_read_server_time = GREATEST(last_read_server_time, sqlc.arg(last_read_server_time))
WHERE room_id = sqlc.arg(room_id) AND user_id = sqlc.arg(user_id);

-- name: RecallMessage :one
UPDATE messages
SET recalled_at = NOW()
WHERE msg_id = sqlc.arg(msg_id)
  AND sender_id = sqlc.arg(sender_id)
  AND recalled_at IS NULL
  AND server_time >= (EXTRACT(EPOCH FROM (NOW() - INTERVAL '2 minutes')) * 1000000)::bigint
RETURNING msg_id, room_id, server_time;
