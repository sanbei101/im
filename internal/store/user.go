package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"
)

func encodeUser(user User) []byte {
	var timestamp [8]byte
	putI64(timestamp[:], user.CreatedAt.UnixMicro())
	data := make([]byte, 0, 16+8+4+len(user.Username)+4+len(user.Password))
	data = putUUID(data, user.UserID)
	data = append(data, timestamp[:]...)
	data = appendString(data, user.Username)
	return appendString(data, user.Password)
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
	if err != nil || !d.done() {
		return User{}, errors.New("invalid user record")
	}
	return User{UserID: id, Username: username, Password: password, CreatedAt: time.UnixMicro(micros)}, nil
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
	if exists, err := s.exists([]byte(usernameKey(username))); err != nil {
		return User{}, fmt.Errorf("check username: %w", err)
	} else if exists {
		return User{}, ErrAlreadyExists
	}
	user := User{UserID: uuid.NewV7(), Username: username, Password: password, CreatedAt: time.Now()}
	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.set(batch, userKey(user.UserID), encodeUser(user)); err != nil {
		return User{}, err
	}
	if err := s.set(batch, usernameKey(username), user.UserID[:]); err != nil {
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
	id, err := s.getUUID([]byte(usernameKey(username)))
	if err != nil {
		return User{}, err
	}
	return getTo(s, []byte(userKey(id)), decodeUser)
}
