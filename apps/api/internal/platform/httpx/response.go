package httpx

import (
	"fmt"
	"log/slog"
	"sort"

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

	// 檔案類錯誤獨立成碼，讓「格式不對」「檔案太大」「範本不符」不再共用
	// VALIDATION_FAILED，使用者才知道要換檔案而不是改欄位。
	CodeUnsupportedFileType    = "UNSUPPORTED_FILE_TYPE"
	CodeFileTooLarge           = "FILE_TOO_LARGE"
	CodeFileUnreadable         = "FILE_UNREADABLE"
	CodeImportTemplateMismatch = "IMPORT_TEMPLATE_MISMATCH"

	CodeRouteNotFound = "ROUTE_NOT_FOUND"

	// 樂觀鎖與衝突裁決類錯誤，讓「資料被別人改過」與「衝突被別人處理過」
	// 不再借用語意不符的 RESOURCE_IN_USE（該碼固定訊息是「無法刪除」）。
	CodeStaleWrite              = "STALE_WRITE"
	CodeConflictAlreadyResolved = "CONFLICT_ALREADY_RESOLVED"
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
	CodeServiceUnavailable: "服務暫時無法使用，請稍後再試",
	CodeResourceInUse:      "此資料已被其他紀錄使用，無法刪除",

	CodeUnsupportedFileType:    "檔案格式不支援，請改用 .xlsx 檔案",
	CodeFileTooLarge:           "檔案超過大小上限，請分批匯入",
	CodeFileUnreadable:         "檔案無法讀取，可能已損毀或非有效的 Excel 檔",
	CodeImportTemplateMismatch: "檔案欄位與匯入範本不符，請下載標準範本重新填寫",

	CodeRouteNotFound: "找不到此功能的服務位址，請重新整理頁面或聯繫系統管理員",

	CodeStaleWrite:              "資料已被其他人更新，請重新整理後再試",
	CodeConflictAlreadyResolved: "此衝突已由其他人處理，請重新整理",
}

// MessageForCode 回傳錯誤碼的預設非技術性訊息；查無對應碼時退回內部系統錯誤訊息。
func MessageForCode(code string) string {
	if message, ok := codeMessages[code]; ok {
		return message
	}
	return codeMessages[CodeInternalError]
}

// ErrorCodes 回傳所有已定義的錯誤碼，供契約測試比對前端字典。
func ErrorCodes() []string {
	codes := make([]string, 0, len(codeMessages))
	for code := range codeMessages {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// APIResponse 定義 API 成功回應結構。
type APIResponse struct {
	Data interface{} `json:"data"`
	Meta interface{} `json:"meta,omitempty"`
}

// ErrorDetail 定義欄位驗證錯誤之詳細資訊。
type ErrorDetail struct {
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason"`
}

// ErrorResponse 定義 API 錯誤回應結構。
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody 定義 API 錯誤主體。
// RequestID 讓使用者回報的畫面訊息能對應到伺服器 log 中同一筆請求。
type ErrorBody struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	Details   []ErrorDetail `json:"details,omitempty"`
	RequestID string        `json:"requestId,omitempty"`
}

// PaginationMeta 定義分頁資訊。
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
// message 一律是呼叫端寫死的非技術性字串；傳空字串代表沿用錯誤碼的預設訊息，
// 避免前端收到空白提示。
func RespondError(c *gin.Context, httpStatus int, code string, message string, details []ErrorDetail) {
	if message == "" {
		message = MessageForCode(code)
	}
	c.AbortWithStatusJSON(httpStatus, ErrorResponse{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: RequestID(c),
		},
	})
}

// logAPIError 僅於伺服器端記錄底層錯誤日誌，避免將系統細節洩漏給前端。
// RespondErrorCode 與 RespondErrorCodeWithReason 共用同一份記錄邏輯，確保兩者的
// 伺服器端可觀測性一致，差別只在於回給前端的 message 來源。
func logAPIError(c *gin.Context, code string, err error) {
	if err == nil {
		return
	}
	slog.Error("api_error",
		slog.String("code", code),
		slog.String("request_id", RequestID(c)),
		slog.String("path", c.Request.URL.Path),
		slog.String("method", c.Request.Method),
		slog.String("error_type", fmt.Sprintf("%T", err)),
		slog.String("error_message", err.Error()),
	)
}

// RespondErrorCode 依錯誤碼查表回傳非技術性錯誤訊息。
func RespondErrorCode(c *gin.Context, httpStatus int, code string, err error, details []ErrorDetail) {
	logAPIError(c, code, err)
	RespondError(c, httpStatus, code, MessageForCode(code), details)
}

// RespondErrorCodeWithReason 與 RespondErrorCode 的伺服器端記錄行為完全相同，
// 差別在於回給前端的 message 改用呼叫端寫死的具體中文 reason（例如「目前密碼不正確」），
// 而不是錯誤碼的通用預設訊息，讓已經知道確切原因的呼叫端不必每次都退回通用句。
// reason 為空字串時退回 MessageForCode(code)，維持與 RespondError 一致的空字串語意。
func RespondErrorCodeWithReason(c *gin.Context, httpStatus int, code string, err error, reason string, details []ErrorDetail) {
	logAPIError(c, code, err)
	RespondError(c, httpStatus, code, reason, details)
}
