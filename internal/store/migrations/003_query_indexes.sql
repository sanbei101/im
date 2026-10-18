CREATE INDEX friend_applications_to_created_idx
    ON friend_applications (to_user_id, created_at DESC);
CREATE INDEX friendships_user_created_idx
    ON friendships (user_id, created_at);
CREATE INDEX blacklist_user_created_idx
    ON blacklist (user_id, created_at);
DROP INDEX IF EXISTS friend_applications_from_idx;
DROP INDEX IF EXISTS users_nickname_lower_idx;

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX users_username_trgm_idx
    ON users USING gin (lower(username) gin_trgm_ops);
CREATE INDEX users_nickname_trgm_idx
    ON users USING gin (lower(nickname) gin_trgm_ops);
