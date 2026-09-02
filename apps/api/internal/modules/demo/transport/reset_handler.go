package transport

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/demo/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// ResetHandler 處理 Demo 資料集重置端點。
type ResetHandler struct {
	svc *app.ResetService
}

// NewResetHandler 建立 ResetHandler 實例。
func NewResetHandler(svc *app.ResetService) *ResetHandler {
	return &ResetHandler{svc: svc}
}

// Reset 處理 POST /api/v1/demo/reset：任何已登入的 Demo 使用者皆可觸發。
func (h *ResetHandler) Reset(c *gin.Context) {
	result, err := h.svc.Reset(c.Request.Context())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeDemoResetFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{
		"datasetVersion": result.DatasetVersion,
		"resetAt":        result.ResetAt.Format(time.RFC3339),
	}, nil)
}
