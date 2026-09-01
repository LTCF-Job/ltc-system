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
	CaseCode    string `json:"caseCode"`
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
	TotalCases   int                     `json:"totalCases"`
	TotalRows    int                     `json:"totalRows"`
	Files        []exportJobFileResponse `json:"files,omitempty"`
	Skipped      []exportJobSkipResponse `json:"skipped,omitempty"`
	ZipFileName  string                  `json:"zipFileName,omitempty"`
	DownloadURL  string                  `json:"downloadUrl,omitempty"`
	ErrorMessage string                  `json:"errorMessage,omitempty"`
	CreatedAt    string                  `json:"createdAt"`
	CompletedAt  string                  `json:"completedAt,omitempty"`
}

// toExportJobResponse 組出單筆工作的完整回應，含逐案下載連結。
func toExportJobResponse(job app.GovClaimJob) exportJobResponse {
	resp := exportJobResponse{
		ID:           job.ID.String(),
		JobType:      job.JobType,
		PeriodYM:     job.PeriodYM,
		Region:       job.Region,
		Mode:         string(job.Mode),
		Status:       job.Status,
		TotalCases:   job.TotalCases,
		TotalRows:    job.TotalRows,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt.Format(time.RFC3339),
	}
	if job.FinishedAt != nil {
		resp.CompletedAt = job.FinishedAt.Format(time.RFC3339)
	}

	for _, file := range job.Files {
		resp.Files = append(resp.Files, exportJobFileResponse{
			CaseID:      file.CaseID.String(),
			CaseCode:    file.CaseCode,
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
			ID:           job.ID.String(),
			JobType:      job.JobType,
			PeriodYM:     job.PeriodYM,
			Region:       job.Region,
			Mode:         string(job.Mode),
			Status:       job.Status,
			TotalCases:   job.TotalCases,
			TotalRows:    job.TotalRows,
			ErrorMessage: job.ErrorMessage,
			CreatedAt:    job.CreatedAt.Format(time.RFC3339),
			CompletedAt:  optionalTime(job.FinishedAt),
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
