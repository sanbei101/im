package store

import (
	"context"
	"uuid"
)

func (s *Store) ReadUsers(
	ctx context.Context,
	roomID, msgID uuid.UUID,
	members []Member,
) ([]uuid.UUID, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	msg, err := s.MessageByID(ctx, roomID, msgID)
	if err != nil {
		return nil, err
	}

	readSeqs, err := s.roomReadSeqs(roomID)
	if err != nil {
		return nil, err
	}

	var readUsers []uuid.UUID
	for _, m := range members {
		if readSeqs[m.UserID] >= msg.RoomSeq {
			readUsers = append(readUsers, m.UserID)
		}
	}
	return readUsers, nil
}
