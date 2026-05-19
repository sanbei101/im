package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/phuslu/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/jwt"
)

// 专为基准测试设计的服务,提供一些批量生成数据的方法
type BenchMockService struct {
	query *db.Queries
}

func NewBenchMockService(query *db.Queries) *BenchMockService {
	return &BenchMockService{query: query}
}

type BenchMockUserInfo struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Token    string    `json:"token"`
}

type BenchMockReq struct {
	SingleRoomNum int   `json:"single_room_num" validate:"min=0,max=10000"`
	GroupRoom     []int `json:"group_room" validate:"dive,min=3,max=10000"`
}

type BatchMockResp struct {
	SingleRooms []struct {
		RoomID string               `json:"room_id"`
		Users []BenchMockUserInfo `json:"users"`
	} `json:"single_rooms"`

	GroupRooms []struct {
		RoomID   string               `json:"room_id"`
		RoomSize int                 `json:"room_size"`
		Users    []BenchMockUserInfo `json:"users"`
	} `json:"group_rooms"`

	TotalUserNum int `json:"total_user_num"`
}

// CreateMock 会在内存中构造所有用户和房间数据,然后进行批量插入
func (s *BenchMockService) CreateMock(ctx context.Context, req BenchMockReq) (*BatchMockResp, error) {
	totalUsers := req.SingleRoomNum * 2
	for _, sz := range req.GroupRoom {
		totalUsers += sz
	}
	if totalUsers == 0 {
		return &BatchMockResp{TotalUserNum: 0}, nil
	}
	resp := &BatchMockResp{TotalUserNum: totalUsers}
	const defaultPassword = "benchpassword"
	hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userParams := make([]db.BatchCreateUserParams, 0, totalUsers)
	users := make([]BenchMockUserInfo, 0, totalUsers)
	for i := 0; i < totalUsers; i++ {
		id := uuid.Must(uuid.NewV7())
		uname := "bench_" + id.String()[:8]
		userParams = append(userParams, db.BatchCreateUserParams{
			UserID:   id,
			Username: uname,
			Password: string(hashed),
		})
		token, err := jwt.GenerateToken(id.String())
		if err != nil {
			log.Error().Err(err).Msgf("failed to generate token for user %s", id)
			return nil, err
		}
		users = append(users, BenchMockUserInfo{UserID: id, Username: uname, Token: token})
	}
	userBR := s.query.BatchCreateUser(ctx, userParams)
	var batchUserErr error
	userBR.Exec(func(i int, e error) {
		if e != nil {
			log.Error().Err(e).Msgf("failed to create user %s", userParams[i].UserID)
			if batchUserErr == nil {
				batchUserErr = e
			}
		}
	})
	if batchUserErr != nil {
		return nil, batchUserErr
	}
	userBR.Close()

	offset := 0
	roomMemberParams := make([]db.BatchCreateRoomMemberParams, 0, totalUsers)

	for i := 0; i < req.SingleRoomNum; i++ {
		if offset+1 >= len(users) {
			break
		}
		u1 := users[offset]
		u2 := users[offset+1]
		offset += 2

		roomID := uuid.Must(uuid.NewV7())
		hash := computeSingleChatHash(u1.UserID, u2.UserID)
		roomName, roomAvatar := generateRoomInfo(roomID)
		roomParams := []db.BatchCreateRoomParams{{
			RoomID:         roomID,
			ChatType:       db.ChatTypeSingle,
			Name:           roomName,
			AvatarUrl:      roomAvatar,
			SingleChatHash: hash,
		}}
		br := s.query.BatchCreateRoom(ctx, roomParams)
		var roomErr error
		br.Exec(func(i int, e error) {
			if e != nil && roomErr == nil {
				log.Error().
					Err(e).
					Msgf("failed to create single chat room for users %s and %s", u1.UserID, u2.UserID)
				roomErr = e
			}
		})
		if roomErr != nil {
			return nil, roomErr
		}
		br.Close()
		roomMemberParams = append(roomMemberParams,
			db.BatchCreateRoomMemberParams{RoomID: roomID, UserID: u1.UserID, Role: db.MemberRoleMember},
			db.BatchCreateRoomMemberParams{RoomID: roomID, UserID: u2.UserID, Role: db.MemberRoleMember},
		)

		resp.SingleRooms = append(
			resp.SingleRooms, struct {
				RoomID string               `json:"room_id"`
				Users []BenchMockUserInfo `json:"users"`
			}{
				RoomID: roomID.String(),
				Users: []BenchMockUserInfo{u1, u2},
			},
		)
	}

	// 群聊房间
	for _, sz := range req.GroupRoom {
		if sz <= 0 {
			continue
		}
		if offset+sz > len(users) {
			sz = len(users) - offset
		}
		if sz <= 0 {
			break
		}
		members := users[offset : offset+sz]
		offset += sz

		roomID := uuid.Must(uuid.NewV7())
		roomName, roomAvatar := generateRoomInfo(roomID)
		roomParams := []db.BatchCreateRoomParams{{
			RoomID:         roomID,
			ChatType:       db.ChatTypeGroup,
			Name:           roomName,
			AvatarUrl:      roomAvatar,
			SingleChatHash: nil,
		}}
		br := s.query.BatchCreateRoom(ctx, roomParams)
		var roomErr error
		br.Exec(func(i int, e error) {
			if e != nil && roomErr == nil {
				roomErr = e
			}
		})
		if roomErr != nil {
			return nil, roomErr
		}
		br.Close()
		for _, mu := range members {
			roomMemberParams = append(roomMemberParams, db.BatchCreateRoomMemberParams{
				RoomID: roomID,
				UserID: mu.UserID,
				Role:   db.MemberRoleMember,
			})
		}

		g := struct {
			RoomID   string               `json:"room_id"`
			RoomSize int                 `json:"room_size"`
			Users    []BenchMockUserInfo `json:"users"`
		}{
			RoomID: roomID.String(),
			RoomSize: sz,
			Users:    members,
		}
		resp.GroupRooms = append(resp.GroupRooms, g)
	}

	if len(roomMemberParams) > 0 {
		memberBR := s.query.BatchCreateRoomMember(ctx, roomMemberParams)
		var memberErr error
		memberBR.Exec(func(i int, e error) {
			if e != nil && memberErr == nil {
				log.Error().Err(e).Msgf("failed to create room member row %d", i)
				memberErr = e
			}
		})
		memberBR.Close()
		if memberErr != nil {
			return nil, memberErr
		}
	}

	return resp, nil
}
