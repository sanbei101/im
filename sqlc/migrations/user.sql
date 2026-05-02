CREATE TABLE users (
    user_id   uuid PRIMARY KEY DEFAULT uuidv7(),
    username  text UNIQUE NOT NULL,
    password  text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users (username);
