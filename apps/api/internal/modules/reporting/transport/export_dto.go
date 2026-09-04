package transport

import (
	"fmt"
	"time"

	"ltc-system/apps/api/internal/modules/reporting/app"
)

// createExportJobRequest 代表建立政府申報匯出工作的請求本體。
// mode 只決定產出的取用方式：direct 逐案下載、zip 打包成單一壓縮檔；一案一檔是唯一顆粒度。
type createExportJobRequest struct {
	JobType  string   `json:"jobType"`
	PeriodYM string   `json:"periodYm" binding:"required"`
	Region   string   `json:"region"`
	Mode     string   `json:"mode" binding:"required,oneof=direct zip"`
	CaseIDs  []string `json:"caseIds" binding:"required,min=1,dive,uuid"`
}

// exportJobFileResponse 代表匯出結果中的單一個案工作簿。
type exportJobFileResponse struct {
	CaseID      string `json:"caseId"`
	CaseName    string `json:"caseName"`
	Region      string `json:"region"`
	RowCount    int    `json:"rowCount"`
	FileName    string `json:"fileName"`
	DownloadURL string `json:"downloadUrl"`
}

// exportJobSkipResponse 代表因資料缺漏未納入申報的趟次統計。
type exportJobSkipResponse struct {
	CaseID   string `json:"caseId"`
	CaseName string `json:"caseName"`
	Reason   string `json:"reason"`
	Count    int    `json:"count"`
}

// exportJobResponse 代表匯出工作的對外形狀。
// skipped 只在建立當下有值：跳過統計不落地，歷史查詢不會重現。
type exportJobResponse struct {
	ID           string                  `json:"id"`
	JobType      string                  `json:"jobType"`
	PeriodYM     string                  `json:"periodYm"`
	Region       string                  `json:"region"`
	Mode         string                  `json:"mode"`
	Status       string                  `json:"status"`
	TotalCases    int                     `json:"totalCases"`
	TotalRows     int                     `json:"totalRows"`
	Files         []exportJobFileResponse `json:"files,omitempty"`
	Skipped       []exportJobSkipResponse `json:"skipped,omitempty"`
	ZipFileName   string                  `json:"zipFileName,omitempty"`
	DownloadURL   string                  `json:"downloadUrl,omitempty"`
	ErrorMessage  string                  `json:"errorMessage,omitempty"`
	CreatedByName string                  `json:"createdByName,omitempty"`
	CreatedAt     string                  `json:"createdAt"`
	CompletedAt   string                  `json:"completedAt,omitempty"`
}

// toExportJobResponse 組出單筆工作的完整回應，含逐案下載連結。
func toExportJobResponse(job app.GovClaimJob) exportJobResponse {
	resp := exportJobResponse{
		ID:            job.ID.String(),
		JobType:       job.JobType,
		PeriodYM:      job.PeriodYM,
		Region:        job.Region,
		Mode:          string(job.Mode),
		Status:        job.Status,
		TotalCases:    job.TotalCases,
		TotalRows:     job.TotalRows,
		ErrorMessage:  job.ErrorMessage,
		CreatedByName: job.CreatedByName,
		CreatedAt:     job.CreatedAt.Format(time.RFC3339),
	}
	if job.FinishedAt != nil {
		resp.CompletedAt = job.FinishedAt.Format(time.RFC3339)
	}

	for _, file := range job.Files {
		resp.Files = append(resp.Files, exportJobFileResponse{
			CaseID:      file.CaseID.String(),
			CaseName:    file.CaseName,
			Region:      file.Region,
			RowCount:    file.RowCount,
			FileName:    file.FileName,
			DownloadURL: caseFileDownloadURL(job.ID.String(), file.CaseID.String()),
		})
	}

	for _, skip := range job.Skipped {
		resp.Skipped = append(resp.Skipped, exportJobSkipResponse{
			CaseID:   skip.CaseID.String(),
			CaseName: skip.CaseName,
			Reason:   skip.Reason,
			Count:    skip.Count,
		})
	}

	// 只有壓縮檔模式才有整包下載；逐案下載模式的連結一律掛在 files 上
	if job.Mode == app.GovClaimModeZip && job.Status == app.ExportStatusSucceeded {
		resp.ZipFileName = app.ZipFileName(job.Region, job.PeriodYM)
		resp.DownloadURL = fmt.Sprintf("/api/v1/exports/%s/download", job.ID.String())
	}

	return resp
}

// toExportJobListResponse 組出歷史列表的回應；列表不帶檔案明細與下載連結。
func toExportJobListResponse(jobs []app.GovClaimJob) []exportJobResponse {
	result := make([]exportJobResponse, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, exportJobResponse{
			ID:            job.ID.String(),
			JobType:       job.JobType,
			PeriodYM:      job.PeriodYM,
			Region:        job.Region,
			Mode:          string(job.Mode),
			Status:        job.Status,
			TotalCases:    job.TotalCases,
			TotalRows:     job.TotalRows,
			ErrorMessage:  job.ErrorMessage,
			CreatedByName: job.CreatedByName,
			CreatedAt:     job.CreatedAt.Format(time.RFC3339),
			CompletedAt:   optionalTime(job.FinishedAt),
		})
	}
	return result
}

func caseFileDownloadURL(jobID, caseID string) string {
	return fmt.Sprintf("/api/v1/exports/%s/files/%s/download", jobID, caseID)
}

func optionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// precheckSummaryResponse 代表前置檢核的統計數字摘要。
type precheckSummaryResponse struct {
	TotalErrors   int `json:"totalErrors"`
	TotalWarnings int `json:"totalWarnings"`
	TotalInfos    int `json:"totalInfos"`
}

// precheckItemDetailResponse 代表單筆檢核項目的問題明細。
type precheckItemDetailResponse struct {
	CaseID      string `json:"caseId,omitempty"`
	CaseName    string `json:"caseName,omitempty"`
	Field       string `json:"field,omitempty"`
	ServiceDate string `json:"serviceDate,omitempty"`
	RideID      string `json:"rideId,omitempty"`
	Description string `json:"description,omitempty"`
}

// precheckItemResponse 代表依規則聚合後的檢核條目。
type precheckItemResponse struct {
	Level   string                       `json:"level"`
	Code    string                       `json:"code"`
	Message string                       `json:"message"`
	Details []precheckItemDetailResponse `json:"details,omitempty"`
}

// precheckResponse 代表前置檢核對外回應形狀，同時滿足前端卡片呈現與向下相容。
type precheckResponse struct {
	Passed        bool                    `json:"passed"`
	HasErrors     bool                    `json:"hasErrors"`
	HasWarnings   bool                    `json:"hasWarnings"`
	TotalErrors   int                     `json:"totalErrors"`
	TotalWarnings int                     `json:"totalWarnings"`
	TotalInfos    int                     `json:"totalInfos"`
	Summary       precheckSummaryResponse `json:"summary"`
	Items         []precheckItemResponse  `json:"items"`
	Issues        []app.PrecheckIssue     `json:"issues,omitempty"`
}

// toPrecheckResponse 將領域層檢核報告轉換為前端所需之結構化 DTO。
func toPrecheckResponse(report *app.PrecheckReport) precheckResponse {
	if report == nil {
		return precheckResponse{
			Passed:  true,
			Summary: precheckSummaryResponse{},
			Items:   []precheckItemResponse{},
		}
	}

	summary := precheckSummaryResponse{
		TotalErrors:   report.TotalErrors,
		TotalWarnings: report.TotalWarnings,
		TotalInfos:    report.TotalInfos,
	}

	type group struct {
		level   string
		code    string
		message string
		details []precheckItemDetailResponse
	}

	var groupOrder []string
	groups := make(map[string]*group)

	for _, issue := range report.Issues {
		code := issue.Code
		g, exists := groups[code]
		if !exists {
			groupOrder = append(groupOrder, code)
			msg := issue.Message
			switch code {
			case "MISSING_CASE_PROFILE":
				msg = "個案基本資料不完整（缺少身分證、住家地址、服務類別或服務使用類型）"
			case "UNRESOLVED_CONFLICT":
				msg = "存在未裁決之混車衝突"
			}
			g = &group{
				level:   string(issue.Severity),
				code:    code,
				message: msg,
			}
			groups[code] = g
		}

		detail := precheckItemDetailResponse{
			Description: issue.Message,
		}
		if issue.Details != nil {
			if cid, ok := issue.Details["caseId"]; ok && cid != nil {
				detail.CaseID = fmt.Sprint(cid)
			}
			if cname, ok := issue.Details["caseName"]; ok && cname != nil {
				detail.CaseName = fmt.Sprint(cname)
			}
			if sdate, ok := issue.Details["serviceDate"]; ok && sdate != nil {
				detail.ServiceDate = fmt.Sprint(sdate)
			}
			if rid, ok := issue.Details["rideId"]; ok && rid != nil {
				detail.RideID = fmt.Sprint(rid)
			}
			if f, ok := issue.Details["field"]; ok && f != nil {
				detail.Field = fmt.Sprint(f)
			} else if code == "MISSING_CASE_PROFILE" {
				detail.Field = "身分證/地址/類別"
			}
		}

		if detail.CaseID != "" || detail.CaseName != "" || detail.RideID != "" || detail.ServiceDate != "" {
			g.details = append(g.details, detail)
		}
	}

	items := make([]precheckItemResponse, 0, len(groupOrder))
	for _, code := range groupOrder {
		g := groups[code]
		items = append(items, precheckItemResponse{
			Level:   g.level,
			Code:    g.code,
			Message: g.message,
			Details: g.details,
		})
	}

	return precheckResponse{
		Passed:        report.Passed,
		HasErrors:     report.TotalErrors > 0,
		HasWarnings:   report.TotalWarnings > 0,
		TotalErrors:   report.TotalErrors,
		TotalWarnings: report.TotalWarnings,
		TotalInfos:    report.TotalInfos,
		Summary:       summary,
		Items:         items,
		Issues:        report.Issues,
	}
}

