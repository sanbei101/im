package store

import (
	"context"
	"errors"
	"fmt"
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
	s.mu.Lock()
	defer s.mu.Unlock()

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
	if err := commit(batch); err != nil {
		return User{}, fmt.Errorf("commit user: %w", err)
	}
	return user, nil
}

func (s *Store) UserByUsername(ctx context.Context, username string) (User, error) {
	if err := contextErr(ctx); err != nil {
		return User{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	var buf [32]byte
	return s.getRecord(appendUserKey(buf[:0], id), decodeUser)
}

func (s *Store) UpdateUserProfile(ctx context.Context, id uuid.UUID, nickname, avatarURL string) (User, error) {
	if err := contextErr(ctx); err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var buf [32]byte
	userKey := appendUserKey(buf[:0], id)
	user, err := s.getRecord(userKey, decodeUser)
	if err != nil {
		return User{}, err
	}

	if strings.TrimSpace(nickname) != "" {
		user.Nickname = strings.TrimSpace(nickname)
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, userKey, encodeUser(user)); err != nil {
		return User{}, err
	}
	if err := commit(batch); err != nil {
		return User{}, fmt.Errorf("commit update user: %w", err)
	}
	return user, nil
}

func (s *Store) SearchUsers(ctx context.Context, keyword string, limit int) ([]User, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := []byte{'u'}
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var results []User
	for iter.First(); iter.Valid() && len(results) < limit; iter.Next() {
		user, err := decodeUser(iter.Value())
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(user.Username), keyword) ||
			strings.Contains(strings.ToLower(user.Nickname), keyword) {
			results = append(results, user)
		}
	}
	return results, iter.Error()
}
