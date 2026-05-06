package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sanbei101/im/internal/api/service"
)

type MessageHandler struct {
	svc *service.MessageService
}

func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

func (h *MessageHandler) GetHistory(c *gin.Context) {
	var req service.HistoryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.svc.GetHistory(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
