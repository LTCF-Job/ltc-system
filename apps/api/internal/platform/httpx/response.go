package httpx

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// 系統統一錯誤碼定義（符合規格書 2.3）
const (
	CodeValidationFailed   = "VALIDATION_FAILED"
	CodeUnauthenticated    = "UNAUTHENTICATED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeAssignmentOverlap  = "ASSIGNMENT_OVERLAP"
	CodeExportInProgress   = "EXPORT_IN_PROGRESS"
	CodePrecheckFailed     = "PRECHECK_FAILED"
	CodeNoExportData       = "NO_EXPORT_DATA"
	CodeMappingRequired    = "MAPPING_REQUIRED"
	CodeReportImportFailed = "DRIVER_REPORT_IMPORT_FAILED"
	CodeFormMappingFailed  = "FORM_MAPPING_FAILED"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeResourceInUse      = "RESOURCE_IN_USE"
)

// codeMessages 為每個錯誤碼提供固定、非技術性的預設訊息，是前端顯示文字的單一事實來源。
// 任何底層（Go、SQL、第三方 SDK）錯誤訊息一律不得回傳給前端，只透過 slog 記錄於伺服器端。
var codeMessages = map[string]string{
	CodeValidationFailed:   "輸入資料不符合規則，請確認後再試",
	CodeUnauthenticated:    "請重新登入",
	CodeForbidden:          "權限不足，無法執行此操作",
	CodeNotFound:           "查無資料",
	CodeAssignmentOverlap:  "該時段已有其他排班，請調整後再試",
	CodeExportInProgress:   "匯出作業進行中，請稍後再試",
	CodePrecheckFailed:     "資料檢核未通過，請確認後再試",
	CodeNoExportData:       "指定條件下沒有可申報的資料",
	CodeMappingRequired:    "尚未完成欄位對應設定",
	CodeReportImportFailed: "匯入司機接送匯報失敗，請確認檔案格式後再試",
	CodeFormMappingFailed:  "更新欄位對應設定失敗，請稍後再試",
	CodeInternalError:      "系統發生錯誤，請稍後再試",
}

// APIResponse 代表成功回應之統一封裝結構。
type APIResponse struct {
	Data interface{} `json:"data"`
	Meta interface{} `json:"meta,omitempty"`
}

// ErrorDetail 代表欄位驗證錯誤之詳細資訊。
type ErrorDetail struct {
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason"`
}

// ErrorResponse 代表失敗回應之統一封裝結構。
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody 代表錯誤本體。
type ErrorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// PaginationMeta 分頁中繼資訊。
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// RespondSuccess 回傳標準成功 JSON 回應。
func RespondSuccess(c *gin.Context, httpStatus int, data interface{}, meta interface{}) {
	c.JSON(httpStatus, APIResponse{
		Data: data,
		Meta: meta,
	})
}

// RespondError 回傳標準錯誤 JSON 回應。
func RespondError(c *gin.Context, httpStatus int, code string, message string, details []ErrorDetail) {
	c.AbortWithStatusJSON(httpStatus, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// RespondErrorCode 依錯誤碼查表回傳統一、非技術性錯誤訊息。
// err 為實際發生的底層錯誤，僅記錄於伺服器端 log，絕不回傳給前端；呼叫端不應再自行組出 err.Error() 作為 message。
func RespondErrorCode(c *gin.Context, httpStatus int, code string, err error, details []ErrorDetail) {
	if err != nil {
		slog.Error("api_error",
			slog.String("code", code),
			slog.String("path", c.Request.URL.Path),
			slog.String("method", c.Request.Method),
			slog.String("error", err.Error()),
		)
	}
	message, ok := codeMessages[code]
	if !ok {
		message = codeMessages[CodeInternalError]
	}
	RespondError(c, httpStatus, code, message, details)
}
