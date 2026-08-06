-- name: AreFriends :one
SELECT EXISTS (
    SELECT 1 FROM friendships
    WHERE user_id_low = sqlc.arg(user_id_low)
      AND user_id_high = sqlc.arg(user_id_high)
);

-- name: IsBlockedBetween :one
SELECT EXISTS (
    SELECT 1 FROM blocks
    WHERE (blocker_id = sqlc.arg(user_a) AND blocked_id = sqlc.arg(user_b))
       OR (blocker_id = sqlc.arg(user_b) AND blocked_id = sqlc.arg(user_a))
);

-- name: UpsertFriendRequest :one
INSERT INTO friend_requests (sender_id, receiver_id)
VALUES (sqlc.arg(sender_id), sqlc.arg(receiver_id))
ON CONFLICT (sender_id, receiver_id) DO UPDATE
SET status = 'pending', updated_at = NOW()
RETURNING request_id, sender_id, receiver_id, status, created_at, updated_at;

-- name: GetFriendRequestForUpdate :one
SELECT request_id, sender_id, receiver_id, status, created_at, updated_at
FROM friend_requests
WHERE request_id = sqlc.arg(request_id)
FOR UPDATE;

-- name: SetFriendRequestStatus :exec
UPDATE friend_requests
SET status = sqlc.arg(status), updated_at = NOW()
WHERE request_id = sqlc.arg(request_id);

-- name: CreateFriendship :exec
INSERT INTO friendships (user_id_low, user_id_high)
VALUES (sqlc.arg(user_id_low), sqlc.arg(user_id_high))
ON CONFLICT DO NOTHING;

-- name: DeleteFriendship :exec
DELETE FROM friendships
WHERE user_id_low = sqlc.arg(user_id_low)
  AND user_id_high = sqlc.arg(user_id_high);

-- name: ListFriends :many
SELECT u.user_id, u.username, u.display_name, u.avatar_url, u.bio
FROM friendships f
JOIN users u ON u.user_id = CASE
    WHEN f.user_id_low = sqlc.arg(user_id) THEN f.user_id_high
    ELSE f.user_id_low
END
WHERE f.user_id_low = sqlc.arg(user_id) OR f.user_id_high = sqlc.arg(user_id)
ORDER BY u.username;

-- name: AddBlock :exec
INSERT INTO blocks (blocker_id, blocked_id)
VALUES (sqlc.arg(blocker_id), sqlc.arg(blocked_id))
ON CONFLICT DO NOTHING;

-- name: DeleteBlock :exec
DELETE FROM blocks
WHERE blocker_id = sqlc.arg(blocker_id) AND blocked_id = sqlc.arg(blocked_id);

-- name: ListBlocks :many
SELECT u.user_id, u.username, u.display_name, u.avatar_url, u.bio
FROM blocks b
JOIN users u ON u.user_id = b.blocked_id
WHERE b.blocker_id = sqlc.arg(blocker_id)
ORDER BY b.created_at DESC;

-- name: ListReceivedFriendRequests :many
SELECT fr.request_id, fr.sender_id, fr.receiver_id, fr.status, fr.created_at,
       u.username, u.display_name, u.avatar_url, u.bio
FROM friend_requests fr
JOIN users u ON u.user_id = fr.sender_id
WHERE fr.receiver_id = sqlc.arg(receiver_id) AND fr.status = 'pending'
ORDER BY fr.created_at DESC;
