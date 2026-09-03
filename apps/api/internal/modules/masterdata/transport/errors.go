package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/platform/httpx"
)

// respondNotFound 以統一錯誤碼回應查無資料。
func respondNotFound(c *gin.Context, message string) {
	httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, message, nil)
}

// respondInvalidID 以統一錯誤碼回應路徑參數格式錯誤。
func respondInvalidID(c *gin.Context, message string) {
	httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, message, nil)
}
