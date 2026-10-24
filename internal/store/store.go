// Package store is the control plane: users, credentials, friendships,
// applications, blacklists, rooms, and members in PostgreSQL. Everything that
// grows with message volume lives in pkg/engine instead.
package store

import (
	"errors"
	"time"
	"uuid"
)

var (
	ErrNotFound      = errors.New("store: not found")
	ErrAlreadyExists = errors.New("store: already exists")
	ErrConflict      = errors.New("store: conflict")
)

type User struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Nickname  string    `json:"nickname"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	ChatTypeSingle = "single"
	ChatTypeGroup  = "group"
)

type Room struct {
	RoomID         uuid.UUID `json:"room_id"`
	ChatType       string    `json:"chat_type"`
	Name           string    `json:"name"`
	AvatarURL      string    `json:"avatar_url"`
	Notice         string    `json:"notice"`
	SingleChatHash []byte    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

type Member struct {
	RoomID   uuid.UUID `json:"room_id"`
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	IsHidden bool      `json:"is_hidden"`
	IsMuted  bool      `json:"is_muted"`
	IsPinned bool      `json:"is_pinned"`
}

type Friend struct {
	UserID    uuid.UUID `json:"user_id"`
	FriendID  uuid.UUID `json:"friend_id"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

type FriendApplication struct {
	FromUserID uuid.UUID `json:"from_user_id"`
	ToUserID   uuid.UUID `json:"to_user_id"`
	Greeting   string    `json:"greeting"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type ConversationInfo struct {
	Room   Room   `json:"room"`
	Member Member `json:"member"`
}

type DeviceInfo struct {
	Token     string    `json:"token"`
	Platform  string    `json:"platform"`
	UpdatedAt time.Time `json:"updated_at"`
}
