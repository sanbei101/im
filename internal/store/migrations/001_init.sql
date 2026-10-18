CREATE TABLE users (
    user_id uuid PRIMARY KEY,
    username text NOT NULL,
    password text NOT NULL,
    nickname text NOT NULL,
    avatar_url text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL
);
CREATE UNIQUE INDEX users_username_lower_key ON users (lower(username));
CREATE INDEX users_nickname_lower_idx ON users (lower(nickname));

CREATE TABLE devices (
    user_id uuid PRIMARY KEY REFERENCES users (user_id) ON DELETE CASCADE,
    token text NOT NULL,
    platform text NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE friendships (
    user_id uuid NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    friend_id uuid NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    remark text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, friend_id),
    CHECK (user_id <> friend_id)
);

CREATE TABLE friend_applications (
    from_user_id uuid NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    to_user_id uuid NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    greeting text NOT NULL DEFAULT '',
    status text NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (to_user_id, from_user_id),
    CHECK (from_user_id <> to_user_id)
);
CREATE INDEX friend_applications_from_idx ON friend_applications (from_user_id);

CREATE TABLE blacklist (
    user_id uuid NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    target_id uuid NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, target_id),
    CHECK (user_id <> target_id)
);

CREATE TABLE rooms (
    room_id uuid PRIMARY KEY,
    chat_type text NOT NULL,
    name text NOT NULL,
    avatar_url text NOT NULL,
    notice text NOT NULL DEFAULT '',
    single_chat_hash bytea,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE UNIQUE INDEX rooms_single_chat_hash_key ON rooms (single_chat_hash) WHERE single_chat_hash IS NOT NULL;

CREATE TABLE room_members (
    room_id uuid NOT NULL REFERENCES rooms (room_id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    role text NOT NULL,
    is_hidden boolean NOT NULL DEFAULT false,
    is_muted boolean NOT NULL DEFAULT false,
    is_pinned boolean NOT NULL DEFAULT false,
    PRIMARY KEY (room_id, user_id)
);
CREATE INDEX room_members_user_idx ON room_members (user_id, room_id);
