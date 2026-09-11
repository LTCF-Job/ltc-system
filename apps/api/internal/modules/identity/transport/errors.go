package transport

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/identity/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// respondIdentityError 依 identity 模組的 sentinel 錯誤映射為對應 HTTP 狀態碼與統一錯誤碼。
func respondIdentityError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, app.ErrIdentityProviderUnconfigured):
		httpx.RespondErrorCode(c, http.StatusServiceUnavailable, httpx.CodeServiceUnavailable, err, nil)
	case errors.Is(err, app.ErrRoleNotFound), errors.Is(err, app.ErrUserNotFound):
		httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
	case errors.Is(err, app.ErrSystemRoleImmutable):
		respondForbiddenWithReason(c, err, "系統內建角色不可修改或刪除")
	case errors.Is(err, app.ErrCannotDeleteSelf):
		respondForbiddenWithReason(c, err, "不可刪除自己的帳號")
	case errors.Is(err, app.ErrCannotResetOwnPassword):
		respondForbiddenWithReason(c, err, "不可用此功能重設自己的密碼，請改用「修改密碼」")
	case errors.Is(err, app.ErrRoleInUse):
		httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeResourceInUse, err, nil)
	case errors.Is(err, app.ErrUnknownRole), errors.Is(err, app.ErrUnknownModuleKey):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
	case errors.Is(err, app.ErrInvalidUserStatus):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
	case errors.Is(err, app.ErrInvalidCredentials):
		httpx.RespondErrorCode(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, err, nil)
	case errors.Is(err, app.ErrEmailAlreadyExists):
		respondValidationWithReason(c, err, "此電子郵件已被使用，請改用其他電子郵件")
	case errors.Is(err, app.ErrWeakPassword):
		respondValidationWithReason(c, err, "密碼強度不足，請使用更複雜的密碼")
	default:
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
	}
}

// respondForbiddenWithReason 記錄底層錯誤後，以呼叫端寫死的具體中文原因回應 403，取代
// httpx.RespondErrorCode 對 FORBIDDEN 的固定通用訊息，讓使用者知道實際原因（角色不可修改／
// 不可刪除自己／不可用此功能重設自己密碼），而不是一律「權限不足」。
func respondForbiddenWithReason(c *gin.Context, err error, reason string) {
	logIdentityError(c, httpx.CodeForbidden, err)
	httpx.RespondError(c, http.StatusForbidden, httpx.CodeForbidden, reason, nil)
}

// respondValidationWithReason 記錄底層錯誤後，以具體中文原因回應 422，用於 Supabase Admin
// API 已辨識出的常見情境（email 已存在、密碼太弱），取代通用的「輸入資料不符合規則」。
func respondValidationWithReason(c *gin.Context, err error, reason string) {
	logIdentityError(c, httpx.CodeValidationFailed, err)
	httpx.RespondError(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, reason, nil)
}

// logIdentityError 比照 httpx.RespondErrorCode 的伺服器端記錄格式，讓改用 httpx.RespondError
// 直接帶入固定訊息的呼叫路徑仍保留可追查的底層錯誤紀錄。
func logIdentityError(c *gin.Context, code string, err error) {
	if err == nil {
		return
	}
	slog.Error("api_error",
		slog.String("code", code),
		slog.String("request_id", httpx.RequestID(c)),
		slog.String("path", c.Request.URL.Path),
		slog.String("method", c.Request.Method),
		slog.String("error_type", fmt.Sprintf("%T", err)),
		slog.String("error_message", err.Error()),
	)
}
