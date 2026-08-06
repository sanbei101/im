CREATE TYPE friend_request_status AS ENUM ('pending', 'accepted', 'rejected');

CREATE TABLE friend_requests (
    request_id uuid PRIMARY KEY DEFAULT uuidv7(),
    sender_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    receiver_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    status friend_request_status NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    CHECK (sender_id <> receiver_id),
    UNIQUE (sender_id, receiver_id)
);

CREATE INDEX idx_friend_requests_receiver_status ON friend_requests (receiver_id, status, updated_at DESC);

CREATE TABLE friendships (
    user_id_low uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    user_id_high uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id_low, user_id_high),
    CHECK (user_id_low < user_id_high)
);

CREATE INDEX idx_friendships_user_id_high ON friendships (user_id_high);

CREATE TABLE blocks (
    blocker_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    blocked_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE INDEX idx_blocks_blocked_id ON blocks (blocked_id);
