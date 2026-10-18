CREATE TABLE room_heads (
    room_id uuid PRIMARY KEY REFERENCES rooms (room_id) ON DELETE CASCADE,
    last_seq bigint NOT NULL DEFAULT 0,
    last_msg jsonb,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE room_reads (
    user_id uuid NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    room_id uuid NOT NULL REFERENCES rooms (room_id) ON DELETE CASCADE,
    read_seq bigint NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, room_id)
);
CREATE INDEX room_reads_room_idx ON room_reads (room_id, user_id);
