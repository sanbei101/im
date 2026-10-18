ALTER TABLE room_heads ADD COLUMN last_msg_time bigint NOT NULL DEFAULT 0;

UPDATE room_heads
SET last_msg_time = COALESCE((last_msg->>'server_time')::bigint, 0)
WHERE last_msg IS NOT NULL;

CREATE INDEX friend_applications_to_created_idx
    ON friend_applications (to_user_id, created_at DESC);
CREATE INDEX friendships_user_created_idx
    ON friendships (user_id, created_at);
CREATE INDEX blacklist_user_created_idx
    ON blacklist (user_id, created_at);
DROP INDEX IF EXISTS friend_applications_from_idx;
DROP INDEX IF EXISTS users_nickname_lower_idx;

DROP INDEX IF EXISTS room_reads_room_idx;
CREATE INDEX room_reads_room_seq_idx
    ON room_reads (room_id, read_seq DESC, user_id);

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX users_username_trgm_idx
    ON users USING gin (lower(username) gin_trgm_ops);
CREATE INDEX users_nickname_trgm_idx
    ON users USING gin (lower(nickname) gin_trgm_ops);
