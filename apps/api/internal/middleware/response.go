package middleware

import (
	"github.com/gin-gonic/gin"
)

// 系統統一錯誤碼定義（符合規格書 2.3）
const (
	CodeValidationFailed    = "VALIDATION_FAILED"
	CodeUnauthenticated     = "UNAUTHENTICATED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeDuplicateNationalID = "DUPLICATE_NATIONAL_ID"
	CodeAssignmentOverlap   = "ASSIGNMENT_OVERLAP"
	CodeExportInProgress    = "EXPORT_IN_PROGRESS"
	CodePrecheckFailed      = "PRECHECK_FAILED"
	CodeMappingRequired     = "MAPPING_REQUIRED"
	CodeIngestTokenInvalid  = "INGEST_TOKEN_INVALID"
	CodeInternalError       = "INTERNAL_ERROR"
)

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
