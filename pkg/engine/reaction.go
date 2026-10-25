package engine

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"uuid"

	"github.com/cockroachdb/pebble"
)

// Reactions live in empty-value keys R:<room>:<seq>:<user>:<emoji>. Each
// reaction is one tiny key insert: concurrent reactors never touch the same
// key, a reaction never rewrites the (potentially kilobytes-large) message
// record, and reactions stay mutable forever even after the message body has
// been frozen into the immutable parquet archive. The key IS the record.

func (e *Engine) AddReaction(ctx context.Context, roomID uuid.UUID, seq uint64, userID uuid.UUID, emoji string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if emoji == "" {
		return errors.New("emoji cannot be empty")
	}

	batch := e.db.NewBatch()
	defer batch.Close()

	key := appendReactionKey(nil, roomID, seq, userID, emoji)
	if err := e.setBytes(batch, key, []byte{}); err != nil {
		return err
	}
	return commit(batch)
}

func (e *Engine) RemoveReaction(
	ctx context.Context,
	roomID uuid.UUID,
	seq uint64,
	userID uuid.UUID,
	emoji string,
) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := e.db.NewBatch()
	defer batch.Close()

	key := appendReactionKey(nil, roomID, seq, userID, emoji)
	if err := batch.Delete(key, nil); err != nil {
		return err
	}
	return commit(batch)
}

// Reactions returns one message's reaction groups, largest group first.
func (e *Engine) Reactions(ctx context.Context, roomID uuid.UUID, seq uint64) ([]ReactionGroup, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	var prefixBuf [reactionPrefixLen]byte
	prefix := appendReactionPrefix(prefixBuf[:0], roomID, seq)
	var upBuf [reactionPrefixLen + 1]byte
	upperBound := prefixUpperBoundBuf(upBuf[:0], prefix)
	iter, err := e.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	return reactionGroups(iter, prefix), iter.Error()
}

// populateReactions attaches each page message's reaction groups. One
// iterator serves the whole page and the messages are visited in seq
// order, so every seek moves forward: the cost tracks what the page shows,
// not what the room accumulated over its lifetime. A room without a single
// reaction key is answered by one probe.
func (e *Engine) populateReactions(ctx context.Context, roomID uuid.UUID, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}
	if err := contextErr(ctx); err != nil {
		return err
	}

	var roomBuf [reactionRoomPrefixLen]byte
	roomPrefix := appendReactionRoomPrefix(roomBuf[:0], roomID)
	var upBuf [reactionRoomPrefixLen + 1]byte
	upperBound := prefixUpperBoundBuf(upBuf[:0], roomPrefix)
	iter, err := e.db.NewIter(&pebble.IterOptions{LowerBound: roomPrefix, UpperBound: upperBound})
	if err != nil {
		return err
	}
	defer iter.Close()
	if !iter.First() {
		return iter.Error()
	}

	order := make([]int, len(messages))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int {
		return cmp.Compare(messages[a].RoomSeq, messages[b].RoomSeq)
	})

	var prefixBuf [reactionPrefixLen]byte
	for _, i := range order {
		prefix := appendReactionPrefix(prefixBuf[:0], roomID, messages[i].RoomSeq)
		if groups := reactionGroups(iter, prefix); len(groups) > 0 {
			messages[i].Reactions = groups
		}
	}
	return iter.Error()
}

// reactionGroups collects one message's reactions through iter: it seeks to
// the message's key prefix and scans until the first key outside it. Keys
// carry no value - the key IS the reaction - so nothing is ever decoded.
func reactionGroups(iter *pebble.Iterator, prefix []byte) []ReactionGroup {
	users := make(map[string][]uuid.UUID)
	prefixLen := len(prefix)
	for iter.SeekGE(prefix); iter.Valid() && bytes.HasPrefix(iter.Key(), prefix); iter.Next() {
		key := iter.Key()
		if len(key) <= prefixLen+16 {
			continue
		}
		var user uuid.UUID
		copy(user[:], key[prefixLen:prefixLen+16])
		emoji := string(key[prefixLen+16:])
		users[emoji] = append(users[emoji], user)
	}
	if len(users) == 0 {
		return nil
	}
	groups := make([]ReactionGroup, 0, len(users))
	for emoji, members := range users {
		groups = append(groups, ReactionGroup{Emoji: emoji, Count: len(members), UserIDs: members})
	}
	slices.SortFunc(groups, func(a, b ReactionGroup) int {
		return cmp.Or(
			cmp.Compare(b.Count, a.Count),
			strings.Compare(a.Emoji, b.Emoji),
		)
	})
	return groups
}
