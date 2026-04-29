-- Active: 1773747183783@@154.8.213.38@5433@database
CREATE TYPE chat_type AS ENUM (
    'single',
    'group'
);

CREATE TYPE message_type AS ENUM (
    'text',
    'image',
    'video',
    'file',
    'system'
);

CREATE TYPE member_role AS ENUM (
    'owner',
    'admin',
    'member'
);

CREATE TABLE rooms (
    room_id uuid PRIMARY KEY,
    chat_type chat_type NOT NULL,
    name VARCHAR(255),
    avatar_url VARCHAR(1024),
    single_chat_hash VARCHAR(64) UNIQUE, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE room_members (
    room_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role member_role NOT NULL DEFAULT 'member',
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    
    PRIMARY KEY (room_id, user_id)
);

CREATE TABLE messages (
    msg_id uuid PRIMARY KEY,
    client_msg_id uuid NOT NULL,
    sender_id uuid NOT NULL,
    room_id uuid NOT NULL,
    chat_type chat_type NOT NULL,
    server_time BIGINT NOT NULL,
    reply_to_msg_id uuid DEFAULT NULL,
    msg_type message_type NOT NULL,
    payload JSONB NOT NULL,
    ext JSONB DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- 用于查询"某个 Room 的所有成员"
CREATE INDEX idx_room_members_user_id ON room_members (user_id);
-- 客户端进入某个 Room 时,按时间倒序拉取历史消息
CREATE INDEX idx_messages_room_time ON messages (room_id, server_time DESC);

-- 用于查询"某个人发过的所有消息"
CREATE INDEX idx_messages_sender_id ON messages (sender_id);