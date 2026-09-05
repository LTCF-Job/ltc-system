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
	BatchMapping(ctx context.Context, updates []app.ColumnMappingUpdate) (int, error)
	MatchPendingColumnsByName(ctx context.Context, name string) ([]app.ColumnMapping, error)
	ListSubmissionReview(ctx context.Context) ([]app.SubmissionReview, error)
	BindPendingDriver(ctx context.Context, driverNameRaw, driverID string) (int, error)
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

// yearMonthPattern 驗證路徑參數格式為西元年月（例如 "2026-03"）。
var yearMonthPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)

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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	form, err := h.svc.CreateForm(c.Request.Context(), req.VehicleID, req.Title)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}
	if form == nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "", nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusCreated, toFormListItemDTO(*form), nil)
}

// DeleteForm 刪除匯報表。
func (h *DriverReportHandler) DeleteForm(c *gin.Context) {
	if err := h.svc.DeleteForm(c.Request.Context(), c.Param("id")); err != nil {
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
		respondImportInputError(c, "file", "僅支援 .xlsx 匯入格式")
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
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	colID := c.Param("id")
	backfilledRows, err := h.svc.UpdateColumnMapping(c.Request.Context(), colID, req.MappingStatus, req.CaseID, req.LegSeq)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	updatedCount, err := h.svc.BatchMapping(c.Request.Context(), req.Mappings)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"updatedCount": updatedCount}, nil)
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
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
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
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	affected, err := h.svc.BindPendingDriver(c.Request.Context(), req.DriverNameRaw, req.DriverID)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"affectedCount": affected}, nil)
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

// respondReportError 把匯入相關的 sentinel 對應到 HTTP 狀態；其餘一律視為匯入失敗。
//
// 匯入失敗多半是使用者自己能修正的表頭或對應問題，因此把原因放進 details 讓前端條列，
// 只顯示通用訊息會讓操作人員無從得知該改哪一欄。
func respondReportError(c *gin.Context, err error) {
	if errors.Is(err, app.ErrFormNotFound) {
		httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "", nil)
		return
	}
	reason := "檔案內容無法解析，請確認為符合範本的 .xlsx 檔案"
	if errors.Is(err, app.ErrInvalidYearMonth) {
		reason = "匯入月份格式錯誤，請使用 YYYY-MM"
	}
	if errors.Is(err, app.ErrImportHasBlockingErrors) {
		reason = "檔案包含阻斷性錯誤，未寫入資料"
	}
	httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeReportImportFailed, err, []httpx.ErrorDetail{
		{Field: "file", Reason: reason},
	})
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
