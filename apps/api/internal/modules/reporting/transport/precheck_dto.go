package transport

import (
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// precheckDetailResponse 對應前端 PrecheckItemDTO.details 的單筆明細。
type precheckDetailResponse struct {
	CaseID      string `json:"caseId,omitempty"`
	CaseName    string `json:"caseName,omitempty"`
	Field       string `json:"field,omitempty"`
	ServiceDate string `json:"serviceDate,omitempty"`
	RideID      string `json:"rideId,omitempty"`
	Description string `json:"description,omitempty"`
}

// precheckItemResponse 對應前端 PrecheckItemDTO。
type precheckItemResponse struct {
	Level   string                   `json:"level"`
	Code    string                   `json:"code"`
	Message string                   `json:"message"`
	Details []precheckDetailResponse `json:"details,omitempty"`
}

// precheckSummaryResponse 對應前端 PrecheckResultDTO.summary。
type precheckSummaryResponse struct {
	TotalErrors   int `json:"totalErrors"`
	TotalWarnings int `json:"totalWarnings"`
	TotalInfos    int `json:"totalInfos"`
}

// precheckResultResponse 對應前端 PrecheckResultDTO；
// app.PrecheckReport 的欄位命名（Issues/TotalErrors）與前端契約（items/hasErrors/summary）不同，需在傳輸層轉換。
type precheckResultResponse struct {
	Passed      bool                    `json:"passed"`
	HasErrors   bool                    `json:"hasErrors"`
	HasWarnings bool                    `json:"hasWarnings"`
	Summary     precheckSummaryResponse `json:"summary"`
	Items       []precheckItemResponse  `json:"items"`
}

// toPrecheckResultResponse 將 PrecheckService 回傳的 PrecheckReport 轉成前端 PrecheckResultDTO 的形狀。
func toPrecheckResultResponse(report *app.PrecheckReport) precheckResultResponse {
	items := make([]precheckItemResponse, 0, len(report.Issues))
	for _, issue := range report.Issues {
		items = append(items, precheckItemResponse{
			Level:   string(issue.Severity),
			Code:    issue.Code,
			Message: issue.Message,
			Details: toPrecheckDetails(issue.Message, issue.Details),
		})
	}

	return precheckResultResponse{
		Passed:      report.Passed,
		HasErrors:   report.TotalErrors > 0,
		HasWarnings: report.TotalWarnings > 0,
		Summary: precheckSummaryResponse{
			TotalErrors:   report.TotalErrors,
			TotalWarnings: report.TotalWarnings,
			TotalInfos:    report.TotalInfos,
		},
		Items: items,
	}
}

// toPrecheckDetails 把 PrecheckIssue.Details 的通用 map 轉成前端固定欄位的明細列。
func toPrecheckDetails(message string, details map[string]interface{}) []precheckDetailResponse {
	if len(details) == 0 {
		return nil
	}
	detail := precheckDetailResponse{Description: message}
	if v, ok := details["caseId"].(string); ok {
		detail.CaseID = v
	}
	if v, ok := details["caseName"].(string); ok {
		detail.CaseName = v
	}
	if v, ok := details["rideId"].(string); ok {
		detail.RideID = v
	}
	if v, ok := details["serviceDate"].(string); ok {
		detail.ServiceDate = v
	}
	return []precheckDetailResponse{detail}
}
