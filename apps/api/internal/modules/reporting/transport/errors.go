package transport

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// respondExportError 將 reporting 的業務錯誤映射為統一的 HTTP 狀態與錯誤碼。
// 這是本模組唯一做這件事的地方，底層 err 只會進 slog，不會回傳給前端。
func respondExportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, app.ErrPrecheckBlocked):
		httpx.RespondErrorCode(c, http.StatusUnprocessableEntity, httpx.CodePrecheckFailed, err, nil)
	case errors.Is(err, app.ErrExportJobNotFound), errors.Is(err, app.ErrExportFileNotFound):
		httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
	case errors.Is(err, app.ErrNoClaimRows):
		httpx.RespondErrorCode(c, http.StatusUnprocessableEntity, httpx.CodeNoExportData, err, nil)
	case errors.Is(err, app.ErrNotZipJob), errors.Is(err, app.ErrInvalidPeriodYM):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
	default:
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
	}
}
