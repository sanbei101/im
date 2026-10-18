package store

import (
	"context"
	"fmt"
	"uuid"
)

func (db *DB) MarkRead(ctx context.Context, userID, roomID uuid.UUID, readSeq uint64) error {
	if _, err := db.pool.Exec(ctx, `
		INSERT INTO room_reads (user_id, room_id, read_seq, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, now())
		ON CONFLICT (user_id, room_id) DO UPDATE
		SET read_seq = GREATEST(room_reads.read_seq, excluded.read_seq),
		    updated_at = now()`,
		userID.String(), roomID.String(), int64(readSeq)); err != nil {
		return fmt.Errorf("mark room read: %w", err)
	}
	return nil
}

func (db *DB) ReadUsers(
	ctx context.Context,
	roomID uuid.UUID,
	messageSeq uint64,
	members []Member,
) ([]uuid.UUID, error) {
	if len(members) == 0 {
		return nil, nil
	}
	raw := make([]string, len(members))
	for i, member := range members {
		raw[i] = member.UserID.String()
	}
	rows, err := db.pool.Query(ctx, `
		SELECT user_id::text FROM room_reads
		WHERE room_id = $1::uuid AND read_seq >= $2 AND user_id = ANY($3::uuid[])`,
		roomID.String(), int64(messageSeq), raw)
	if err != nil {
		return nil, fmt.Errorf("load read users: %w", err)
	}
	defer rows.Close()
	var result []uuid.UUID
	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			return nil, err
		}
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
