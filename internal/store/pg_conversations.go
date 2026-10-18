package store

import (
	"context"
	"fmt"
	"uuid"
)

// Conversations returns the rooms a user belongs to plus their read cursor.
// Message previews, latest sequence and unread counts are deliberately absent:
// clients maintain their own conversation state and lazy-load visible rooms.
func (db *DB) Conversations(ctx context.Context, userID uuid.UUID) ([]ConversationInfo, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT r.room_id::text, r.chat_type, r.name, r.avatar_url, r.notice,
		       r.single_chat_hash, r.created_at, r.updated_at,
		       m.room_id::text, m.user_id::text, m.role, m.is_hidden, m.is_muted, m.is_pinned
		FROM room_members m
		JOIN rooms r ON r.room_id = m.room_id
		WHERE m.user_id = $1::uuid
		ORDER BY r.updated_at DESC, r.room_id`, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	result := make([]ConversationInfo, 0, 16)
	for rows.Next() {
		var (
			room         Room
			member       Member
			roomID       string
			memberRoomID string
			memberUserID string
		)
		if err := rows.Scan(
			&roomID, &room.ChatType, &room.Name, &room.AvatarURL, &room.Notice,
			&room.SingleChatHash, &room.CreatedAt, &room.UpdatedAt,
			&memberRoomID, &memberUserID, &member.Role,
			&member.IsHidden, &member.IsMuted, &member.IsPinned,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		if room.RoomID, err = uuid.Parse(roomID); err != nil {
			return nil, fmt.Errorf("parse conversation room_id: %w", err)
		}
		if member.RoomID, err = uuid.Parse(memberRoomID); err != nil {
			return nil, fmt.Errorf("parse conversation member room_id: %w", err)
		}
		if member.UserID, err = uuid.Parse(memberUserID); err != nil {
			return nil, fmt.Errorf("parse conversation user_id: %w", err)
		}
		result = append(result, ConversationInfo{Room: room, Member: member})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	return result, nil
}
