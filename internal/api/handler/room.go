package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sanbei101/im/internal/api/middleware"
	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/internal/api/validate"
)

type RoomHandler struct {
	svc *service.RoomService
}

func NewRoomHandler(svc *service.RoomService) *RoomHandler {
	return &RoomHandler{svc: svc}
}

func (h *RoomHandler) CreateOrGetSingleChatRoom(c *gin.Context) {
	var req service.CreateRoomReq
	err := validate.ValidateAndParseJSON(c, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	resp, err := h.svc.CreateOrGetSingleChatRoom(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *RoomHandler) CreateGroupRoom(c *gin.Context) {
	var req service.CreateGroupRoomReq
	err := validate.ValidateAndParseJSON(c, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.CreateGroupRoom(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *RoomHandler) ListRooms(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := h.svc.ListRooms(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *RoomHandler) BatchCreateRooms(c *gin.Context) {
	var req service.BatchCreateRoomsReq
	err := validate.ValidateAndParseJSON(c, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.BatchCreateRooms(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}
