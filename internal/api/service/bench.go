package service

import (
	"github.com/google/uuid"
	"github.com/sanbei101/im/internal/db"
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
		Users []BenchMockUserInfo `json:"users"`
	} `json:"single_rooms"`

	GroupRooms []struct {
		RoomSize int                 `json:"room_size"`
		Users    []BenchMockUserInfo `json:"users"`
	} `json:"group_rooms"`

	TotalUserNum int `json:"total_user_num"`
}
