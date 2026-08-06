ALTER TABLE users
    ADD COLUMN display_name text NOT NULL DEFAULT '',
    ADD COLUMN avatar_url text NOT NULL DEFAULT '',
    ADD COLUMN bio text NOT NULL DEFAULT '',
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT NOW();

CREATE TABLE refresh_sessions (
    session_id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_sessions_user_id ON refresh_sessions (user_id);
CREATE INDEX idx_refresh_sessions_expires_at ON refresh_sessions (expires_at);
