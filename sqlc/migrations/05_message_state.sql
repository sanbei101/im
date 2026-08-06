ALTER TABLE room_members
    ADD COLUMN last_read_server_time BIGINT NOT NULL DEFAULT 0;

ALTER TABLE messages
    ADD COLUMN recalled_at TIMESTAMPTZ;

CREATE INDEX idx_messages_room_unread
    ON messages (room_id, server_time);
