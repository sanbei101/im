package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/internal/api/validate"
)

type MessageHandler struct {
	svc *service.MessageService
}

func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

func (h *MessageHandler) GetHistory(c *gin.Context) {
	var req service.HistoryReq
	err := validate.ValidateAndParseQuery(c, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.GetHistory(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
