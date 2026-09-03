package app

import (
	"context"
	"fmt"
)

// PrecheckRepositoryPort 定義前置檢核資料查詢介面。
type PrecheckRepositoryPort interface {
	FindIncompleteActiveCases(ctx context.Context, region string) ([]IncompleteCase, error)
	FindUnresolvedConflicts(ctx context.Context, region string) ([]UnresolvedConflict, error)
}

// PrecheckSeverity 代表檢核結果等級。
type PrecheckSeverity string

const (
	SeverityError   PrecheckSeverity = "error"
	SeverityWarning PrecheckSeverity = "warning"
	SeverityInfo    PrecheckSeverity = "info"
)

// PrecheckIssue 代表單筆檢核項目結果。
type PrecheckIssue struct {
	Severity PrecheckSeverity       `json:"severity"`
	Code     string                 `json:"code"`
	Message  string                 `json:"message"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// PrecheckReport 前置檢核完整報告。
type PrecheckReport struct {
	Passed        bool            `json:"passed"` // 無 error 時為 true
	TotalErrors   int             `json:"totalErrors"`
	TotalWarnings int             `json:"totalWarnings"`
	TotalInfos    int             `json:"totalInfos"`
	Issues        []PrecheckIssue `json:"issues"`
}

// PrecheckService 負責在匯出前執行全方位資料驗證。
type PrecheckService struct {
	repo PrecheckRepositoryPort
}

// NewPrecheckService 建立 PrecheckService 實例。
func NewPrecheckService(repo PrecheckRepositoryPort) *PrecheckService {
	return &PrecheckService{repo: repo}
}

// RunPrecheck 執行指定月份與區域之申報前置檢核（規格書 7.6）。
func (s *PrecheckService) RunPrecheck(ctx context.Context, periodYM string, region string) (*PrecheckReport, error) {
	var issues []PrecheckIssue
	errorCount := 0
	warningCount := 0
	infoCount := 0

	// 1. 固定 Info: 配給額度檢查未執行（決議）
	issues = append(issues, PrecheckIssue{
		Severity: SeverityInfo,
		Code:     "QUOTA_CHECK_SKIPPED",
		Message:  "個案配給額度檢查未執行——尚未取得額度計算規則",
	})
	infoCount++

	if s.repo != nil {
		// 2. 檢查是否有個案缺必要欄位 (身分證、住址、使用類型)
		if incompleteCases, err := s.repo.FindIncompleteActiveCases(ctx, region); err == nil {
			for _, c := range incompleteCases {
				issues = append(issues, PrecheckIssue{
					Severity: SeverityError,
					Code:     "MISSING_CASE_PROFILE",
					Message:  fmt.Sprintf("個案「%s」缺少身分證、住家地址、服務類別或服務使用類型", c.Name),
					Details: map[string]interface{}{
						"caseId":   c.ID,
						"caseName": c.Name,
					},
				})
				errorCount++
			}
		}

		// 3. 檢查是否有未處理的混車衝突
		if conflicts, err := s.repo.FindUnresolvedConflicts(ctx, region); err == nil {
			for _, conf := range conflicts {
				issues = append(issues, PrecheckIssue{
					Severity: SeverityWarning,
					Code:     "UNRESOLVED_CONFLICT",
					Message:  fmt.Sprintf("個案「%s」於 %s 存在未裁決之混車衝突", conf.CaseName, conf.ServiceDate.Format("2006-01-02")),
					Details: map[string]interface{}{
						"rideId":      conf.RideID,
						"caseName":    conf.CaseName,
						"serviceDate": conf.ServiceDate.Format("2006-01-02"),
					},
				})
				warningCount++
			}
		}
	}

	return &PrecheckReport{
		Passed:        errorCount == 0,
		TotalErrors:   errorCount,
		TotalWarnings: warningCount,
		TotalInfos:    infoCount,
		Issues:        issues,
	}, nil
}
