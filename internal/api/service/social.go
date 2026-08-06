package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
)

type FriendService struct {
	query *db.Queries
	db    *pgxpool.Pool
}

func NewFriendService(query *db.Queries, dbPool *pgxpool.Pool) *FriendService {
	return &FriendService{query: query, db: dbPool}
}

type UserIDReq struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

type FriendRequestResp struct {
	RequestID  string    `json:"request_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Status     string    `json:"status"`
	Sender     *UserResp `json:"sender,omitempty"`
}

var (
	ErrCannotRelateSelf    = errors.New("cannot create relation with self")
	ErrUserBlocked         = errors.New("user relationship is blocked")
	ErrAlreadyFriends      = errors.New("users are already friends")
	ErrFriendRequestExists = errors.New("friend request already pending")
	ErrFriendRequestAbsent = errors.New("friend request not found")
	ErrFriendRequestDenied = errors.New("friend request access denied")
	ErrFriendRequestClosed = errors.New("friend request is not pending")
)

func (s *FriendService) SendRequest(ctx context.Context, userID string, req UserIDReq) (*FriendRequestResp, error) {
	sender, receiver, err := relationUsers(userID, req.UserID)
	if err != nil {
		return nil, err
	}
	if _, err := s.query.GetUserByID(ctx, receiver); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get friend request receiver: %w", err)
	}
	blocked, err := s.query.IsBlockedBetween(ctx, db.IsBlockedBetweenParams{UserA: sender, UserB: receiver})
	if err != nil {
		return nil, fmt.Errorf("check blocks: %w", err)
	}
	if blocked {
		return nil, ErrUserBlocked
	}
	low, high := orderedUsers(sender, receiver)
	friends, err := s.query.AreFriends(ctx, db.AreFriendsParams{UserIDLow: low, UserIDHigh: high})
	if err != nil {
		return nil, fmt.Errorf("check friendship: %w", err)
	}
	if friends {
		return nil, ErrAlreadyFriends
	}
	request, err := s.query.UpsertFriendRequest(
		ctx,
		db.UpsertFriendRequestParams{SenderID: sender, ReceiverID: receiver},
	)
	if err != nil {
		return nil, fmt.Errorf("upsert friend request: %w", err)
	}
	return friendRequestResponse(request), nil
}

func (s *FriendService) ListReceivedRequests(ctx context.Context, userID string) ([]*FriendRequestResp, error) {
	user, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	rows, err := s.query.ListReceivedFriendRequests(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("list received friend requests: %w", err)
	}
	result := make([]*FriendRequestResp, 0, len(rows))
	for _, row := range rows {
		result = append(result, &FriendRequestResp{
			RequestID:  row.RequestID.String(),
			SenderID:   row.SenderID.String(),
			ReceiverID: row.ReceiverID.String(),
			Status:     string(row.Status),
			Sender:     userResponse(row.SenderID, row.Username, row.DisplayName, row.AvatarUrl, row.Bio),
		})
	}
	return result, nil
}

func (s *FriendService) AcceptRequest(ctx context.Context, userID, requestID string) error {
	user, request, tx, err := s.lockRequest(ctx, userID, requestID)
	if err != nil {
		return err
	}
	committed := false
	defer rollbackTx(ctx, tx, &committed, "friend request accept")
	if request.ReceiverID != user {
		return ErrFriendRequestDenied
	}
	if request.Status != db.FriendRequestStatusPending {
		return ErrFriendRequestClosed
	}
	blocked, err := s.query.WithTx(tx).
		IsBlockedBetween(ctx, db.IsBlockedBetweenParams{UserA: request.SenderID, UserB: request.ReceiverID})
	if err != nil {
		return fmt.Errorf("check friend request blocks: %w", err)
	}
	if blocked {
		return ErrUserBlocked
	}
	q := s.query.WithTx(tx)
	if err := q.SetFriendRequestStatus(
		ctx,
		db.SetFriendRequestStatusParams{RequestID: request.RequestID, Status: db.FriendRequestStatusAccepted},
	); err != nil {
		return fmt.Errorf("accept friend request: %w", err)
	}
	low, high := orderedUsers(request.SenderID, request.ReceiverID)
	if err := q.CreateFriendship(ctx, db.CreateFriendshipParams{UserIDLow: low, UserIDHigh: high}); err != nil {
		return fmt.Errorf("create friendship: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit friend request accept: %w", err)
	}
	committed = true
	return nil
}

func (s *FriendService) RejectRequest(ctx context.Context, userID, requestID string) error {
	user, request, tx, err := s.lockRequest(ctx, userID, requestID)
	if err != nil {
		return err
	}
	committed := false
	defer rollbackTx(ctx, tx, &committed, "friend request reject")
	if request.ReceiverID != user {
		return ErrFriendRequestDenied
	}
	if request.Status != db.FriendRequestStatusPending {
		return ErrFriendRequestClosed
	}
	if err := s.query.WithTx(tx).
		SetFriendRequestStatus(ctx, db.SetFriendRequestStatusParams{RequestID: request.RequestID, Status: db.FriendRequestStatusRejected}); err != nil {
		return fmt.Errorf("reject friend request: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit friend request reject: %w", err)
	}
	committed = true
	return nil
}

func (s *FriendService) DeleteFriend(ctx context.Context, userID, friendID string) error {
	user, friend, err := relationUsers(userID, friendID)
	if err != nil {
		return err
	}
	low, high := orderedUsers(user, friend)
	if err := s.query.DeleteFriendship(ctx, db.DeleteFriendshipParams{UserIDLow: low, UserIDHigh: high}); err != nil {
		return fmt.Errorf("delete friendship: %w", err)
	}
	return nil
}

func (s *FriendService) ListFriends(ctx context.Context, userID string) ([]*UserResp, error) {
	user, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	rows, err := s.query.ListFriends(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("list friends: %w", err)
	}
	return publicUsers(rows), nil
}

func (s *FriendService) Block(ctx context.Context, userID string, req UserIDReq) error {
	blocker, blocked, err := relationUsers(userID, req.UserID)
	if err != nil {
		return err
	}
	if _, err := s.query.GetUserByID(ctx, blocked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("get blocked user: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin block transaction: %w", err)
	}
	committed := false
	defer rollbackTx(ctx, tx, &committed, "block user")
	q := s.query.WithTx(tx)
	low, high := orderedUsers(blocker, blocked)
	if err := q.DeleteFriendship(ctx, db.DeleteFriendshipParams{UserIDLow: low, UserIDHigh: high}); err != nil {
		return fmt.Errorf("delete friendship while blocking: %w", err)
	}
	if err := q.AddBlock(ctx, db.AddBlockParams{BlockerID: blocker, BlockedID: blocked}); err != nil {
		return fmt.Errorf("add block: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit block transaction: %w", err)
	}
	committed = true
	return nil
}

func (s *FriendService) Unblock(ctx context.Context, userID, blockedID string) error {
	blocker, blocked, err := relationUsers(userID, blockedID)
	if err != nil {
		return err
	}
	if err := s.query.DeleteBlock(ctx, db.DeleteBlockParams{BlockerID: blocker, BlockedID: blocked}); err != nil {
		return fmt.Errorf("delete block: %w", err)
	}
	return nil
}

func (s *FriendService) ListBlocks(ctx context.Context, userID string) ([]*UserResp, error) {
	user, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	rows, err := s.query.ListBlocks(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("list blocks: %w", err)
	}
	result := make([]*UserResp, 0, len(rows))
	for _, row := range rows {
		result = append(result, userResponse(row.UserID, row.Username, row.DisplayName, row.AvatarUrl, row.Bio))
	}
	return result, nil
}

func (s *FriendService) lockRequest(
	ctx context.Context,
	userID, requestID string,
) (uuid.UUID, *db.FriendRequest, pgx.Tx, error) {
	user, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, nil, nil, ErrInvalidInput
	}
	requestUUID, err := uuid.Parse(requestID)
	if err != nil {
		return uuid.Nil, nil, nil, ErrInvalidInput
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, nil, nil, fmt.Errorf("begin friend request transaction: %w", err)
	}
	request, err := s.query.WithTx(tx).GetFriendRequestForUpdate(ctx, requestUUID)
	if err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			log.Error().Err(rollbackErr).Msg("failed to rollback missing friend request transaction")
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, nil, nil, ErrFriendRequestAbsent
		}
		return uuid.Nil, nil, nil, fmt.Errorf("get friend request: %w", err)
	}
	return user, request, tx, nil
}

func relationUsers(left, right string) (uuid.UUID, uuid.UUID, error) {
	first, err := uuid.Parse(left)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	second, err := uuid.Parse(right)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	if first == second {
		return uuid.Nil, uuid.Nil, ErrCannotRelateSelf
	}
	return first, second, nil
}

func orderedUsers(left, right uuid.UUID) (uuid.UUID, uuid.UUID) {
	if bytes.Compare(left[:], right[:]) < 0 {
		return left, right
	}
	return right, left
}

func friendRequestResponse(request *db.FriendRequest) *FriendRequestResp {
	return &FriendRequestResp{
		RequestID:  request.RequestID.String(),
		SenderID:   request.SenderID.String(),
		ReceiverID: request.ReceiverID.String(),
		Status:     string(request.Status),
	}
}

func publicUsers(rows []*db.ListFriendsRow) []*UserResp {
	result := make([]*UserResp, 0, len(rows))
	for _, row := range rows {
		result = append(result, userResponse(row.UserID, row.Username, row.DisplayName, row.AvatarUrl, row.Bio))
	}
	return result
}

func rollbackTx(ctx context.Context, tx pgx.Tx, committed *bool, operation string) {
	if !*committed {
		if err := tx.Rollback(ctx); err != nil {
			log.Error().Err(err).Str("operation", operation).Msg("failed to rollback transaction")
		}
	}
}
