package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

func encodeUser(user User) []byte {
	var timestamp [8]byte
	putI64(timestamp[:], user.CreatedAt.UnixMicro())
	data := make([]byte, 0, 16+8+4+len(user.Username)+4+len(user.Password)+4+len(user.Nickname)+4+len(user.AvatarURL))
	data = putUUID(data, user.UserID)
	data = append(data, timestamp[:]...)
	data = appendString(data, user.Username)
	data = appendString(data, user.Password)
	data = appendString(data, user.Nickname)
	return appendString(data, user.AvatarURL)
}

func decodeUser(data []byte) (User, error) {
	d := decoder{data: data}
	id, err := d.uuid()
	if err != nil {
		return User{}, err
	}
	micros, err := d.i64()
	if err != nil {
		return User{}, err
	}
	username, err := d.string()
	if err != nil {
		return User{}, err
	}
	password, err := d.string()
	if err != nil {
		return User{}, err
	}
	nickname, err := d.string()
	if err != nil {
		return User{}, err
	}
	avatar, err := d.string()
	if err != nil || !d.done() {
		return User{}, errors.New("invalid user record")
	}
	return User{
		UserID:    id,
		Username:  username,
		Password:  password,
		Nickname:  nickname,
		AvatarURL: avatar,
		CreatedAt: time.UnixMicro(micros),
	}, nil
}

func (s *Store) CreateUser(ctx context.Context, username, password string) (User, error) {
	if err := contextErr(ctx); err != nil {
		return User{}, err
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return User{}, errors.New("username is required")
	}

	var buf [64]byte
	unameKey := appendUsernameKey(buf[:0], username)
	exists, err := s.exists(unameKey)
	if err != nil {
		return User{}, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return User{}, ErrAlreadyExists
	}

	user := User{
		UserID:    uuid.NewV7(),
		Username:  username,
		Password:  password,
		Nickname:  username,
		AvatarURL: "",
		CreatedAt: time.Now(),
	}
	batch := s.db.NewBatch()
	defer batch.Close()

	var uKeyBuf [32]byte
	if err := s.setBytes(batch, appendUserKey(uKeyBuf[:0], user.UserID), encodeUser(user)); err != nil {
		return User{}, err
	}
	if err := s.setBytes(batch, unameKey, user.UserID[:]); err != nil {
		return User{}, err
	}
	// The nickname defaults to the username; index it from day one so
	// nickname search finds every user.
	nicknameKey := appendNicknameKey(nil, strings.ToLower(user.Nickname), user.UserID)
	if err := s.setBytes(batch, nicknameKey, user.UserID[:]); err != nil {
		return User{}, err
	}
	if err := commit(batch); err != nil {
		return User{}, fmt.Errorf("commit user: %w", err)
	}
	return user, nil
}

func (s *Store) UserByUsername(ctx context.Context, username string) (User, error) {
	if err := contextErr(ctx); err != nil {
		return User{}, err
	}

	var buf [64]byte
	id, err := s.getRecord(appendUsernameKey(buf[:0], username), getUUID)
	if err != nil {
		return User{}, err
	}
	var uKeyBuf [32]byte
	return s.getRecord(appendUserKey(uKeyBuf[:0], id), decodeUser)
}

func (s *Store) UserByID(ctx context.Context, id uuid.UUID) (User, error) {
	if err := contextErr(ctx); err != nil {
		return User{}, err
	}

	var buf [32]byte
	return s.getRecord(appendUserKey(buf[:0], id), decodeUser)
}

func (s *Store) UpdateUserProfile(ctx context.Context, id uuid.UUID, nickname, avatarURL string) (User, error) {
	if err := contextErr(ctx); err != nil {
		return User{}, err
	}

	var buf [32]byte
	userKey := appendUserKey(buf[:0], id)
	user, err := s.getRecord(userKey, decodeUser)
	if err != nil {
		return User{}, err
	}

	var oldNicknameKey []byte
	if trimmed := strings.TrimSpace(nickname); trimmed != "" && trimmed != user.Nickname {
		oldNicknameKey = appendNicknameKey(nil, strings.ToLower(user.Nickname), id)
		user.Nickname = trimmed
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, userKey, encodeUser(user)); err != nil {
		return User{}, err
	}
	if oldNicknameKey != nil {
		if err := batch.Delete(oldNicknameKey, nil); err != nil {
			return User{}, err
		}
		newNicknameKey := appendNicknameKey(nil, strings.ToLower(user.Nickname), id)
		if err := s.setBytes(batch, newNicknameKey, id[:]); err != nil {
			return User{}, err
		}
	}
	if err := commit(batch); err != nil {
		return User{}, fmt.Errorf("commit update user: %w", err)
	}
	return user, nil
}

// SearchUsers matches keyword against the lowercase name indexes: 'n' for
// usernames, 'N' for nicknames. Scanning index keys avoids decoding every
// user record; only the matches pay for a record read.
func (s *Store) SearchUsers(ctx context.Context, keyword string, limit int) ([]User, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, nil
	}
	kw := []byte(strings.ToLower(keyword))

	seen := make(map[uuid.UUID]struct{}, limit)
	ids := make([]uuid.UUID, 0, limit)
	if parsedID, err := uuid.Parse(keyword); err == nil {
		if _, err := s.UserByID(ctx, parsedID); err == nil {
			seen[parsedID] = struct{}{}
			ids = append(ids, parsedID)
		}
	}
	if len(ids) < limit {
		if err := s.matchUsersByUsername(kw, limit, seen, &ids); err != nil {
			return nil, err
		}
	}
	if len(ids) < limit {
		if err := s.matchUsersByNickname(kw, limit, seen, &ids); err != nil {
			return nil, err
		}
	}

	usersByID, err := s.UsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(ids))
	for _, id := range ids {
		if user, ok := usersByID[id]; ok {
			users = append(users, user)
		}
	}
	return users, nil
}

// matchUsersByUsername scans the 'n' index: key 'n'+lower(username), value the
// user id. Caller provides the seen set to dedupe users matching both indexes.
func (s *Store) matchUsersByUsername(kw []byte, limit int, seen map[uuid.UUID]struct{}, ids *[]uuid.UUID) error {
	var buf [1]byte
	prefix := appendUsernamePrefix(buf[:0])
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
	if err != nil {
		return err
	}
	defer iter.Close()
	for iter.First(); iter.Valid() && len(*ids) < limit; iter.Next() {
		if !containsFold(iter.Key()[1:], kw) {
			continue
		}
		var id uuid.UUID
		copy(id[:], iter.Value())
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		*ids = append(*ids, id)
	}
	return iter.Error()
}

// matchUsersByNickname scans the 'N' index: key 'N'+lower(nickname)+user id.
func (s *Store) matchUsersByNickname(kw []byte, limit int, seen map[uuid.UUID]struct{}, ids *[]uuid.UUID) error {
	var buf [1]byte
	prefix := appendNicknamePrefix(buf[:0])
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
	if err != nil {
		return err
	}
	defer iter.Close()
	for iter.First(); iter.Valid() && len(*ids) < limit; iter.Next() {
		key := iter.Key()
		if !containsFold(key[1:len(key)-16], kw) {
			continue
		}
		var id uuid.UUID
		copy(id[:], key[len(key)-16:])
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		*ids = append(*ids, id)
	}
	return iter.Error()
}

// UsersByIDs resolves user records in one iterator: ids are seeked in sorted
// order instead of one Get per id. Missing ids are simply absent from the map.
func (s *Store) UsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]User, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	sorted := slices.Clone(ids)
	slices.SortFunc(sorted, func(a, b uuid.UUID) int { return a.Compare(b) })
	sorted = slices.Compact(sorted)

	var buf [1]byte
	prefix := appendUserPrefix(buf[:0])
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	users := make(map[uuid.UUID]User, len(sorted))
	var keyBuf [17]byte
	for _, id := range sorted {
		key := appendUserKey(keyBuf[:0], id)
		if !iter.SeekGE(key) || !bytes.Equal(iter.Key(), key) {
			continue
		}
		user, err := decodeUser(iter.Value())
		if err != nil {
			return nil, err
		}
		users[id] = user
	}
	return users, iter.Error()
}

func (s *Store) UpdateUserPassword(ctx context.Context, id uuid.UUID, password string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var buf [32]byte
	userKey := appendUserKey(buf[:0], id)
	user, err := s.getRecord(userKey, decodeUser)
	if err != nil {
		return err
	}

	user.Password = password

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, userKey, encodeUser(user)); err != nil {
		return err
	}
	if err := commit(batch); err != nil {
		return fmt.Errorf("commit update password: %w", err)
	}
	return nil
}
