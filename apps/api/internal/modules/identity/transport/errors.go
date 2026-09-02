package transport

import (
	"errors"
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
	case errors.Is(err, app.ErrSystemRoleImmutable), errors.Is(err, app.ErrCannotDeleteSelf):
		httpx.RespondErrorCode(c, http.StatusForbidden, httpx.CodeForbidden, err, nil)
	case errors.Is(err, app.ErrRoleInUse):
		httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeResourceInUse, err, nil)
	case errors.Is(err, app.ErrUnknownRole):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
	case errors.Is(err, app.ErrInvalidCredentials):
		httpx.RespondErrorCode(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, err, nil)
	default:
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
	}
}
