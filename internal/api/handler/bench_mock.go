package handler

import (
	"net/http"

	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/pkg/render"
)

type BenchMockHandler struct {
	svc *service.BenchMockService
}

func NewBenchMockHandler(svc *service.BenchMockService) *BenchMockHandler {
	return &BenchMockHandler{svc: svc}
}

func (h *BenchMockHandler) CreateMock(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.BenchMockReq](w, r)
	if err != nil {
		return
	}

	resp, err := h.svc.CreateMock(r.Context(), req)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, "failed to create bench mock")
		return
	}

	render.Success(w, "批量造数成功", resp)
}
