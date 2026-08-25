package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/repository"
	"ltc-system/apps/api/internal/service"
)

// CaseHandler 處理個案相關之 HTTP 請求。
type CaseHandler struct {
	caseRepo      *repository.CaseRepository
	masterService *service.MasterService
	importService *service.ImportService
}

// NewCaseHandler 建立 CaseHandler 實例。
func NewCaseHandler(
	caseRepo *repository.CaseRepository,
	masterService *service.MasterService,
	importService *service.ImportService,
) *CaseHandler {
	return &CaseHandler{
		caseRepo:      caseRepo,
		masterService: masterService,
		importService: importService,
	}
}

// List 查詢個案清單（回傳遮罩身分證）。
func (h *CaseHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	region := c.Query("region")
	status := c.Query("status")
	q := c.Query("q")

	cases, total, err := h.caseRepo.List(c.Request.Context(), region, status, q, page, pageSize)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢個案失敗", nil)
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	middleware.RespondSuccess(c, http.StatusOK, cases, middleware.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// Create 新增個案主檔。
func (h *CaseHandler) Create(c *gin.Context) {
	var req service.CreateCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	entity, err := h.masterService.CreateCase(
		c.Request.Context(), req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		if err == service.ErrDuplicateNationalID {
			middleware.RespondError(c, http.StatusConflict, middleware.CodeDuplicateNationalID, "身分證字號重複", nil)
			return
		}
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, entity, nil)
}

// Reveal 解密個案身分證號。
func (h *CaseHandler) Reveal(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	plainID, err := h.masterService.RevealCaseNationalID(
		c.Request.Context(), id, actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "個案不存在或解密失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"nationalId": plainID}, nil)
}

// CreateSchedule 建立個案排班設定與時段明細。
func (h *CaseHandler) CreateSchedule(c *gin.Context) {
	var req service.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	sched, err := h.masterService.CreateCaseSchedule(c.Request.Context(), req)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, sched, nil)
}

// ImportExcel 批次上傳解析個案新增資料.xlsx。
func (h *CaseHandler) ImportExcel(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "未提供上傳檔案", nil)
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無法開啟檔案", nil)
		return
	}
	defer f.Close()

	preview, err := h.importService.ParseCasesFromExcel(f)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, preview, nil)
}
