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
	data := []byte{recordVersion}
	var id [16]byte
	putUUID(id[:], user.UserID)
	data = append(data, id[:]...)
	var timestamp [8]byte
	putI64(timestamp[:], user.CreatedAt.UnixMicro())
	data = append(data, timestamp[:]...)
	data = appendString(data, user.Username)
	return appendString(data, user.Password)
}

func decodeUser(data []byte) (User, error) {
	if len(data) < 1 || data[0] != recordVersion {
		return User{}, errors.New("invalid user record version")
	}
	d := decoder{data: data[1:]}
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
	if _, closer, err := s.db.Get([]byte(usernameKey(username))); err == nil {
		closer.Close()
		return User{}, ErrAlreadyExists
	} else if !errors.Is(err, pebble.ErrNotFound) {
		return User{}, fmt.Errorf("check username: %w", err)
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
	var id uuid.UUID
	if err := s.get([]byte(usernameKey(username)), func(value []byte) error {
		var err error
		id, err = getUUID(value)
		return err
	}); err != nil {
		return User{}, err
	}
	var user User
	if err := s.get([]byte(userKey(id)), func(value []byte) error {
		var err error
		user, err = decodeUser(value)
		return err
	}); err != nil {
		return User{}, err
	}
	return user, nil
}
