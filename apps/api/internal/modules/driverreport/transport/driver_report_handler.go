package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/driverreport/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// DriverReportServiceInterface 定義 DriverReportHandler 所需的業務服務介面。
type DriverReportServiceInterface interface {
	ListForms(ctx context.Context) ([]app.ReportForm, error)
	ListImportedMonths(ctx context.Context) ([]app.ImportedMonth, error)
	GetMonthDetail(ctx context.Context, formID uuid.UUID, yearMonth string) (*app.MonthDetail, error)
	CreateForm(ctx context.Context, vehicleID, title string) (*app.ReportForm, error)
	DeleteForm(ctx context.Context, formID string) error
	ListColumns(ctx context.Context, formID, mappingStatus string) ([]app.ColumnMapping, error)
	UpdateColumnMapping(ctx context.Context, colID, status string, caseID *string, legSeq *int16) (int, error)
	BatchMapping(ctx context.Context, updates []app.ColumnMappingUpdate) (*app.BatchMappingResult, error)
	MatchPendingColumnsByName(ctx context.Context, name string) ([]app.ColumnMapping, error)
	ListSubmissionReview(ctx context.Context) ([]app.SubmissionReview, error)
	BindPendingDriver(ctx context.Context, driverNameRaw, driverID string) (int, error)
	ResolveRowConflict(ctx context.Context, conflictID string, useNew bool, actor app.Actor) error
	IgnoreColumn(ctx context.Context, colID string, actor app.Actor) error
	IgnoreRowConflict(ctx context.Context, conflictID string, actor app.Actor) error
	IgnoreSubmission(ctx context.Context, submissionID string, actor app.Actor) error
	TemplateExcel(ctx context.Context, formID uuid.UUID) ([]byte, string, error)
	ParseDriverReport(ctx context.Context, formID uuid.UUID, r io.Reader, yearMonth string) (*app.PreviewResult, error)
	CommitDriverReport(ctx context.Context, formID uuid.UUID, r io.Reader, decisions []app.ColumnDecision, yearMonth string, actor app.Actor) (*app.CommitResult, error)
}

// DriverReportHandler 處理司機接送匯報表的登記、匯入與欄位對應之 HTTP 請求。
type DriverReportHandler struct {
	svc DriverReportServiceInterface
}

// NewDriverReportHandler 建立 DriverReportHandler 實例。
func NewDriverReportHandler(svc DriverReportServiceInterface) *DriverReportHandler {
	return &DriverReportHandler{svc: svc}
}

// ListForms 取得所有車輛的匯報表清單。
func (h *DriverReportHandler) ListForms(c *gin.Context) {
	forms, err := h.svc.ListForms(c.Request.Context())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	items := make([]FormListItemDTO, 0, len(forms))
	for _, f := range forms {
		items = append(items, toFormListItemDTO(f))
	}
	httpx.RespondSuccess(c, http.StatusOK, items, nil)
}

// ListImportedMonths 取得每份匯報表各月份已匯入的筆數與最後匯入時間。
func (h *DriverReportHandler) ListImportedMonths(c *gin.Context) {
	months, err := h.svc.ListImportedMonths(c.Request.Context())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	items := make([]ImportedMonthDTO, 0, len(months))
	for _, m := range months {
		items = append(items, toImportedMonthDTO(m))
	}
	httpx.RespondSuccess(c, http.StatusOK, items, nil)
}

// yearMonthPattern 驗證路徑參數格式為西元年月（例如 "2026-03"）；月份限定 01-12，
// 讓「2026-13」這類格式對但月份不合法的輸入在這裡就被擋下回 400，不會通過正則
// 後才在 rocdate.MonthRangeStrict 失敗、被當成非預期系統錯誤回 500。
var yearMonthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// GetMonthDetail 取得某份匯報表指定月份已匯入的完整內容：逐日回報明細與展開後的個案搭乘紀錄。
func (h *DriverReportHandler) GetMonthDetail(c *gin.Context) {
	formID, ok := parseFormID(c)
	if !ok {
		return
	}

	yearMonth := c.Param("yearMonth")
	if !yearMonthPattern.MatchString(yearMonth) {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "月份格式錯誤，應為 YYYY-MM", nil)
		return
	}

	detail, err := h.svc.GetMonthDetail(c.Request.Context(), formID, yearMonth)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, toMonthDetailDTO(*detail), nil)
}

// CreateForm 為一台車建立匯報表。
func (h *DriverReportHandler) CreateForm(c *gin.Context) {
	var req CreateFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	form, err := h.svc.CreateForm(c.Request.Context(), req.VehicleID, req.Title)
	if err != nil {
		if errors.Is(err, app.ErrVehicleRequired) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "請選擇車輛", nil)
			return
		}
		if app.IsInputError(err) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "輸入資料不符合規則，請確認後再試", []httpx.ErrorDetail{
				{Reason: err.Error()},
			})
			return
		}
		// 其餘（建立/查詢匯報表時的資料庫錯誤）是系統問題，不該跟輸入驗證共用 400。
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}
	if form == nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "建立匯報表失敗，請稍後再試", nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusCreated, toFormListItemDTO(*form), nil)
}

// DeleteForm 刪除匯報表。ID 格式錯誤與「已不存在」統一映射 404，不再讓格式錯誤落到
// 500，也不再讓刪除不存在的表單悄悄回 200 成功（repo 層已改為檢查 RowsAffected）。
func (h *DriverReportHandler) DeleteForm(c *gin.Context) {
	if err := h.svc.DeleteForm(c.Request.Context(), c.Param("id")); err != nil {
		if errors.Is(err, app.ErrFormNotFound) {
			httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "查無此匯報表，可能已被刪除，請重新整理後再試", nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"success": true}, nil)
}

// DownloadTemplate 下載該車匯報表的空白範本 (.xlsx)。
func (h *DriverReportHandler) DownloadTemplate(c *gin.Context) {
	formID, ok := parseFormID(c)
	if !ok {
		return
	}

	excelBytes, vehicleName, err := h.svc.TemplateExcel(c.Request.Context(), formID)
	if err != nil {
		respondReportError(c, err)
		return
	}

	attachAs(c, "driver_report_template.xlsx", fmt.Sprintf("%s接送匯報範本.xlsx", vehicleName))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

// ImportExcel 上傳匯報表 .xlsx；dryRun=true 回傳預覽，dryRun=false 正式寫入。
//
// yearMonth（YYYY-MM）為選填的宣告匯入月份：有帶時整個月會被這份檔案覆蓋；檔案內
// 出現該月以外的日期不會整份拒絕，只有那幾列會標成錯誤列並在寫入時被跳過。
// 未帶時只覆蓋檔案實際涵蓋的日期。
func (h *DriverReportHandler) ImportExcel(c *gin.Context) {
	formID, ok := parseFormID(c)
	if !ok {
		return
	}

	fileHeader, ok := httpx.BindUploadFile(c, "file")
	if !ok {
		return
	}
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".xlsx") {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeUnsupportedFileType, errors.New("unsupported driver report file extension"), []httpx.ErrorDetail{
			{Field: "file", Reason: "僅支援 .xlsx 匯入格式"},
		})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		respondImportInputError(c, "file", "無法開啟檔案")
		return
	}
	defer f.Close()

	yearMonth := c.Query("yearMonth")

	// 依 dryRun 參數區分預覽或正式寫入
	if c.DefaultQuery("dryRun", "true") == "false" {
		decisions, err := parseColumnDecisions(c.PostForm("columnDecisions"))
		if err != nil {
			respondImportInputError(c, "columnDecisions", "欄位對應資料格式錯誤，請重新上傳檔案")
			return
		}

		result, err := h.svc.CommitDriverReport(c.Request.Context(), formID, f, decisions, yearMonth, app.Actor{
			ActorID:   auth.GetActorID(c),
			ActorRole: auth.GetActorRole(c),
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		if err != nil {
			respondReportError(c, err)
			return
		}
		httpx.RespondSuccess(c, http.StatusOK, result, nil)
		return
	}

	preview, err := h.svc.ParseDriverReport(c.Request.Context(), formID, f, yearMonth)
	if err != nil {
		respondReportError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, preview, nil)
}

// ListColumns 取得欄位對應清單。
func (h *DriverReportHandler) ListColumns(c *gin.Context) {
	cols, err := h.svc.ListColumns(c.Request.Context(), c.Query("formId"), c.Query("mappingStatus"))
	if err != nil {
		// 純查詢端點沒有使用者可修正的輸入，失敗一律是系統/資料庫問題，不該沿用
		// 「更新欄位對應設定失敗」這句只適用於寫入端點的訊息。
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	items := make([]FormColumnDTO, 0, len(cols))
	for _, col := range cols {
		items = append(items, toFormColumnDTO(col))
	}
	httpx.RespondSuccess(c, http.StatusOK, items, nil)
}

// UpdateColumnMapping 綁定或略過單一欄位對應。
func (h *DriverReportHandler) UpdateColumnMapping(c *gin.Context) {
	var req UpdateColumnMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	colID := c.Param("id")
	backfilledRows, err := h.svc.UpdateColumnMapping(c.Request.Context(), colID, req.MappingStatus, req.CaseID, req.LegSeq)
	if err != nil {
		respondColumnMappingError(c, err)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{
		"id":             colID,
		"mappingStatus":  req.MappingStatus,
		"caseId":         req.CaseID,
		"legSeq":         req.LegSeq,
		"backfilledRows": backfilledRows,
	}, nil)
}

// BatchMapping 批次更新多個欄位對應。
func (h *DriverReportHandler) BatchMapping(c *gin.Context) {
	var req BatchMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	result, err := h.svc.BatchMapping(c.Request.Context(), req.Mappings)
	if err != nil {
		respondColumnMappingError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, result, nil)
}

// MatchPendingColumnsByName 找出待維護欄位中姓名與傳入姓名相符（含近似）的欄位，供新
// 建個案後主動詢問使用者這批欄位是否也是同一人。
func (h *DriverReportHandler) MatchPendingColumnsByName(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		httpx.RespondSuccess(c, http.StatusOK, []FormColumnDTO{}, nil)
		return
	}

	cols, err := h.svc.MatchPendingColumnsByName(c.Request.Context(), name)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	items := make([]FormColumnDTO, 0, len(cols))
	for _, col := range cols {
		items = append(items, toFormColumnDTO(col))
	}
	httpx.RespondSuccess(c, http.StatusOK, items, nil)
}

// ListSubmissionReview 以匯報表列為單位列出待維護資料，一列可能同時有個案欄位與駕駛
// 人兩種問題。
func (h *DriverReportHandler) ListSubmissionReview(c *gin.Context) {
	reviews, err := h.svc.ListSubmissionReview(c.Request.Context())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	items := make([]SubmissionReviewDTO, 0, len(reviews))
	for _, r := range reviews {
		items = append(items, toSubmissionReviewDTO(r))
	}
	httpx.RespondSuccess(c, http.StatusOK, items, nil)
}

// BindDriver 把某個比對不到司機主檔的原始姓名綁定到指定司機，回填既有回報已寫入的
// 搭乘紀錄，不需要重新上傳原始檔案。
func (h *DriverReportHandler) BindDriver(c *gin.Context) {
	var req BindDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	affected, err := h.svc.BindPendingDriver(c.Request.Context(), req.DriverNameRaw, req.DriverID)
	if err != nil {
		respondColumnMappingError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"affectedCount": affected}, nil)
}

// ResolveRowConflict 裁決一筆「同車同個案」衝突：useNew 採用這次上傳的新值，
// 否則保留既有資料，兩者都只標記這筆衝突已解決。
func (h *DriverReportHandler) ResolveRowConflict(c *gin.Context) {
	var req ResolveRowConflictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	err := h.svc.ResolveRowConflict(c.Request.Context(), c.Param("id"), req.UseNew, app.Actor{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		respondRowConflictError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"success": true}, nil)
}

// IgnoreColumn 忽略一筆欄位對應待維護資料，直接把該列從系統刪除。
func (h *DriverReportHandler) IgnoreColumn(c *gin.Context) {
	err := h.svc.IgnoreColumn(c.Request.Context(), c.Param("id"), app.Actor{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, app.ErrPendingItemNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		respondColumnMappingError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}

// IgnoreRowConflict 忽略一筆「同車同個案」衝突，直接把該衝突列從系統刪除。
func (h *DriverReportHandler) IgnoreRowConflict(c *gin.Context) {
	err := h.svc.IgnoreRowConflict(c.Request.Context(), c.Param("id"), app.Actor{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, app.ErrPendingItemNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		respondColumnMappingError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}

// IgnoreSubmission 忽略一筆駕駛人未比對到司機主檔的匯報列，直接把該列從系統刪除。
func (h *DriverReportHandler) IgnoreSubmission(c *gin.Context) {
	err := h.svc.IgnoreSubmission(c.Request.Context(), c.Param("id"), app.Actor{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, app.ErrPendingItemNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		respondColumnMappingError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}

func parseFormID(c *gin.Context) (uuid.UUID, bool) {
	formID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "匯報表 ID 格式錯誤", nil)
		return uuid.Nil, false
	}
	return formID, true
}

// respondImportInputError 將匯入請求本身的可修正問題放入 details，讓批次頁能在對應檔案列顯示原因。
func respondImportInputError(c *gin.Context, field, reason string) {
	httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, errors.New(reason), []httpx.ErrorDetail{
		{Field: field, Reason: reason},
	})
}

// respondReportError 依實際原因分流匯入／下載範本失敗的狀態碼與訊息，供 ImportExcel
// 與 DownloadTemplate 共用：查無表單→404；月份格式／欄位對應輸入問題→400 附具體原因；
// 檔案太大／檔案損毀／表頭找不到範本各自的碼；其餘（DB 故障、範本產生失敗等）→500。
// 不再把所有原因都壓進 DRIVER_REPORT_IMPORT_FAILED 並統一顯示「請確認檔案格式」——
// 範本下載失敗時這句話尤其誤導，因為使用者根本沒有上傳檔案可以檢查格式。
func respondReportError(c *gin.Context, err error) {
	if errors.Is(err, app.ErrFormNotFound) {
		httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "查無此匯報表，可能已被刪除，請重新整理後再試", nil)
		return
	}
	if errors.Is(err, app.ErrInvalidYearMonth) {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "匯入月份格式錯誤，請使用 YYYY-MM", nil)
		return
	}
	if app.IsInputError(err) {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "輸入資料不符合規則，請確認後再試", []httpx.ErrorDetail{
			{Field: "file", Reason: err.Error()},
		})
		return
	}

	// 底下用訊息內容分流：這幾類錯誤都是本模組自己（excel.go／parse.go／
	// platform/spreadsheet）寫出的固定中文描述，不是 pgx 或其他底層 SDK 的原文，
	// 拿來當 details reason 顯示是安全的。
	msg := err.Error()
	switch {
	case strings.Contains(msg, "超過上限"):
		httpx.RespondErrorCode(c, http.StatusRequestEntityTooLarge, httpx.CodeFileTooLarge, err, nil)
	case strings.Contains(msg, "找不到「") || strings.Contains(msg, "找不到匯報表表頭"):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeImportTemplateMismatch, err, []httpx.ErrorDetail{
			{Field: "file", Reason: msg},
		})
	case strings.Contains(msg, "zip 結構無效") ||
		strings.Contains(msg, "ZIP 項目") ||
		strings.Contains(msg, "開啟 Excel 檔案失敗") ||
		strings.Contains(msg, "無工作表資料") ||
		strings.Contains(msg, "讀取上傳檔案失敗"):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeFileUnreadable, err, nil)
	default:
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
	}
}

// respondColumnMappingError 是欄位對應寫入類端點（UpdateColumnMapping／BatchMapping／
// BindDriver／Ignore*）共用的錯誤映射：使用者可修正的輸入或業務規則問題
// （app.IsInputError）回 400 並帶出具體原因，真正的系統/資料庫錯誤才回 500，不再讓
// 兩種情況都套上同一句「更新欄位對應設定失敗」。
func respondColumnMappingError(c *gin.Context, err error) {
	if app.IsInputError(err) {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "輸入資料不符合規則，請確認後再試", []httpx.ErrorDetail{
			{Reason: err.Error()},
		})
		return
	}
	httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
}

// respondRowConflictError 把「同車同個案」衝突已被他人裁決過的情況映射成 409，
// 而不是與其他未預期錯誤共用 500；其餘錯誤交回 respondColumnMappingError 判斷輸入
// 或系統原因。
func respondRowConflictError(c *gin.Context, err error) {
	if errors.Is(err, app.ErrRowConflictAlreadyResolved) {
		httpx.RespondError(c, http.StatusConflict, httpx.CodeResourceInUse, "此衝突已由其他人處理，請重新整理", nil)
		return
	}
	respondColumnMappingError(c, err)
}

// parseColumnDecisions 解析使用者於預覽階段就地確認的欄位對應（JSON 陣列）；
// 空字串代表沒有新的對應決定，既有對應維持不變。
func parseColumnDecisions(raw string) ([]app.ColumnDecision, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var decisions []app.ColumnDecision
	if err := json.Unmarshal([]byte(raw), &decisions); err != nil {
		return nil, err
	}
	return decisions, nil
}

// attachAs 同時給出 ASCII 後備檔名與 UTF-8 檔名，讓舊瀏覽器不致收到亂碼。
func attachAs(c *gin.Context, asciiName, utf8Name string) {
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", asciiName, url.PathEscape(utf8Name)))
}
