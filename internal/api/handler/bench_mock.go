package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/internal/api/validate"
)

type BenchMockHandler struct {
	svc *service.BenchMockService
}

func NewBenchMockHandler(svc *service.BenchMockService) *BenchMockHandler {
	return &BenchMockHandler{svc: svc}
}

func (h *BenchMockHandler) CreateMock(c *gin.Context) {
	var req service.BenchMockReq
	if err := validate.ValidateAndParseJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.CreateMock(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}
