package transport

import (
	"fmt"
	"net/url"
	"time"

	"ltc-system/apps/api/internal/modules/reporting/app"
)

// createExportJobRequest 代表建立政府申報匯出工作的請求本體。
// mode 只決定產出的取用方式：direct 逐案下載、zip 打包成單一壓縮檔；一案一檔是唯一顆粒度。
type createExportJobRequest struct {
	JobType  string   `json:"jobType"`
	PeriodYM string   `json:"periodYm" binding:"required"`
	Mode     string   `json:"mode" binding:"required,oneof=direct zip"`
	CaseIDs  []string `json:"caseIds" binding:"required,min=1,dive,uuid"`
}

// createRegionExportRequest 代表依區域批次建立申報匯出的請求本體。
//
// 刻意與 createExportJobRequest 分開而不是把欄位加上去：逐案勾選流程完全不動，
// 回歸風險為零；BindJSONStrict 也會擋掉送錯端點的多餘欄位。
// periodYms 上限 12 是為了控制同步產檔的量體（一區域一個月可能就是數十份檔案）。
type createRegionExportRequest struct {
	Regions   []string `json:"regions" binding:"required,min=1,dive,min=1"`
	PeriodYMs []string `json:"periodYms" binding:"required,min=1,max=12,dive,len=5"`
}

// siteTripSummaryRequest 代表據點趟數彙總表的請求本體。
type siteTripSummaryRequest struct {
	SiteIDs   []string `json:"siteIds" binding:"required,min=1,dive,uuid"`
	PeriodYMs []string `json:"periodYms" binding:"required,min=1,max=12,dive,len=5"`
}

// regionExportMonthResponse 代表批次匯出中單一月份的結果。
type regionExportMonthResponse struct {
	PeriodYM     string             `json:"periodYm"`
	Succeeded    bool               `json:"succeeded"`
	ErrorMessage string             `json:"errorMessage,omitempty"`
	Job          *exportJobResponse `json:"job,omitempty"`
}

// regionExportResultResponse 代表依區域批次匯出的整體結果。
type regionExportResultResponse struct {
	Regions          []string                    `json:"regions"`
	PeriodYMs        []string                    `json:"periodYms"`
	CaseCount        int                         `json:"caseCount"`
	TotalFiles       int                         `json:"totalFiles"`
	Months           []regionExportMonthResponse `json:"months"`
	BatchDownloadURL string                      `json:"batchDownloadUrl,omitempty"`
	BatchFileName    string                      `json:"batchFileName,omitempty"`
}

// exportJobFileResponse 代表匯出結果中的單一個案工作簿。
type exportJobFileResponse struct {
	CaseID      string `json:"caseId"`
	CaseName    string `json:"caseName"`
	RowCount    int    `json:"rowCount"`
	FileName    string `json:"fileName"`
	DownloadURL string `json:"downloadUrl"`
}

// exportJobDataGapResponse 代表因來源缺漏而在申報檔留白的欄位統計。
type exportJobDataGapResponse struct {
	CaseID   string `json:"caseId"`
	CaseName string `json:"caseName"`
	Reason   string `json:"reason"`
	Count    int    `json:"count"`
}

// exportJobResponse 代表匯出工作的對外形狀。
// dataGaps 只在建立當下有值：缺漏統計不落地，歷史查詢不會重現。
type exportJobResponse struct {
	ID            string                     `json:"id"`
	JobType       string                     `json:"jobType"`
	PeriodYM      string                     `json:"periodYm"`
	Mode          string                     `json:"mode"`
	Status        string                     `json:"status"`
	TotalCases    int                        `json:"totalCases"`
	TotalRows     int                        `json:"totalRows"`
	Files         []exportJobFileResponse    `json:"files,omitempty"`
	DataGaps      []exportJobDataGapResponse `json:"dataGaps,omitempty"`
	ZipFileName   string                     `json:"zipFileName,omitempty"`
	DownloadURL   string                     `json:"downloadUrl,omitempty"`
	ErrorMessage  string                     `json:"errorMessage,omitempty"`
	CreatedByName string                     `json:"createdByName,omitempty"`
	CreatedAt     string                     `json:"createdAt"`
	CompletedAt   string                     `json:"completedAt,omitempty"`
}

// toExportJobResponse 組出單筆工作的完整回應，含逐案下載連結。
func toExportJobResponse(job app.GovClaimJob) exportJobResponse {
	resp := exportJobResponse{
		ID:            job.ID.String(),
		JobType:       job.JobType,
		PeriodYM:      job.PeriodYM,
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
			RowCount:    file.RowCount,
			FileName:    file.FileName,
			DownloadURL: caseFileDownloadURL(job.ID.String(), file.CaseID.String()),
		})
	}

	for _, gap := range job.DataGaps {
		resp.DataGaps = append(resp.DataGaps, exportJobDataGapResponse{
			CaseID:   gap.CaseID.String(),
			CaseName: gap.CaseName,
			Reason:   gap.Reason,
			Count:    gap.Count,
		})
	}

	// 只有壓縮檔模式才有整包下載；逐案下載模式的連結一律掛在 files 上
	if job.Mode == app.GovClaimModeZip && job.Status == app.ExportStatusSucceeded {
		resp.ZipFileName = app.ZipFileName(job.PeriodYM)
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

// toRegionExportResultResponse 組出批次匯出的回應，含跨月合併下載連結。
func toRegionExportResultResponse(result app.RegionClaimResult) regionExportResultResponse {
	resp := regionExportResultResponse{
		Regions:    result.Regions,
		PeriodYMs:  result.PeriodYMs,
		CaseCount:  result.CaseCount,
		TotalFiles: result.TotalFiles,
		Months:     make([]regionExportMonthResponse, 0, len(result.Months)),
	}

	for _, month := range result.Months {
		item := regionExportMonthResponse{
			PeriodYM:     month.PeriodYM,
			Succeeded:    month.Succeeded,
			ErrorMessage: month.ErrorMessage,
		}
		if month.Succeeded {
			job := toExportJobResponse(month.Job)
			item.Job = &job
		}
		resp.Months = append(resp.Months, item)
	}

	if len(result.SucceededJobIDs) > 0 {
		query := url.Values{}
		for _, id := range result.SucceededJobIDs {
			query.Add("jobIds", id.String())
		}
		resp.BatchDownloadURL = "/api/v1/exports/batch-download?" + query.Encode()
		resp.BatchFileName = app.BatchZipFileName(result.PeriodYMs)
	}

	return resp
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
