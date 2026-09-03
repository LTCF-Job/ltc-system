package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/platform/httpx"
)

// respondInvalidID 以統一錯誤碼回應路徑參數格式錯誤。
func respondInvalidID(c *gin.Context, message string) {
	httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, message, nil)
}
