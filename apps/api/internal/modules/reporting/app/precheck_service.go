package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// PrecheckRepositoryPort 定義前置檢核資料查詢介面。
type PrecheckRepositoryPort interface {
	FindIncompleteActiveCases(ctx context.Context, scope ClaimScope) ([]IncompleteCase, error)
	FindUnresolvedConflicts(ctx context.Context, scope ClaimScope) ([]UnresolvedConflict, error)
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

// RunPrecheck 執行指定月份之申報前置檢核（規格書 7.6）。
//
// 目前只有兩項檢核：個案資料缺漏（warning，不擋）與未裁決混車衝突（error，擋）。
// 規格書列的其他檢核項目（例如個案配給額度）尚未實作，也不會出現在報告中——這裡原本有
// 一則恆常輸出的 QUOTA_CHECK_SKIPPED info，但每次都出現、永遠不會變的提示對操作者沒有
// 資訊量，已移除。infoCount 保留是為了維持 TotalInfos 的對外契約。
func (s *PrecheckService) RunPrecheck(ctx context.Context, scope ClaimScope) (*PrecheckReport, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("precheck repository is not configured")
	}

	var issues []PrecheckIssue
	errorCount := 0
	warningCount := 0
	infoCount := 0

	// 1. 檢查是否有個案缺必要欄位 (身分證、住址、使用類型)。
	// 缺資料只留白該欄位，不阻擋匯出；擋下整批會讓其餘報得出來的資料一起報不出去。
	incompleteCases, err := s.repo.FindIncompleteActiveCases(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("find incomplete active cases: %w", err)
	}
	for _, c := range incompleteCases {
		issues = append(issues, PrecheckIssue{
			Severity: SeverityWarning,
			Code:     "MISSING_CASE_PROFILE",
			Message:  fmt.Sprintf("個案「%s」缺少身分證、住家地址、服務類別或服務使用類型，該欄位將留白匯出", c.Name),
			Details: map[string]interface{}{
				"caseId":   c.ID,
				"caseName": c.Name,
			},
		})
		warningCount++
	}

	// 2. 未裁決衝突（混車）會使申報來源不確定，是唯一仍阻擋匯出的檢核項目：
	// 缺資料只是欄位留白，混車卻會讓報出去的資料本身是錯的。
	conflicts, err := s.repo.FindUnresolvedConflicts(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("find unresolved conflicts: %w", err)
	}
	for _, conf := range conflicts {
		issues = append(issues, PrecheckIssue{
			Severity: SeverityError,
			Code:     "UNRESOLVED_CONFLICT",
			Message:  fmt.Sprintf("個案「%s」於 %s 存在未裁決之混車衝突", conf.CaseName, conf.ServiceDate.Format("2006-01-02")),
			Details: map[string]interface{}{
				"rideId":      conf.RideID,
				"caseName":    conf.CaseName,
				"serviceDate": conf.ServiceDate.Format("2006-01-02"),
			},
		})
		errorCount++
	}

	return &PrecheckReport{
		Passed:        errorCount == 0,
		TotalErrors:   errorCount,
		TotalWarnings: warningCount,
		TotalInfos:    infoCount,
		Issues:        issues,
	}, nil
}

// RunPrecheckMulti 對多個月份各跑一次檢核後合併成單一報告。
//
// 兩種項目的合併方式刻意不同：
//   - MISSING_CASE_PROFILE 講的是個案主檔缺欄位，與月份無關，跨月會重複命中同一個案，
//     因此依 caseId 去重，否則選 6 個月就會看到同一個案的同一則警告出現 6 次。
//   - UNRESOLVED_CONFLICT 綁定特定日期的特定搭乘紀錄，每一筆都要各自裁決，全部保留。
//
// 逐月項目一律在 details 補上 periodYm，否則使用者看不出是哪一個月出問題。
func (s *PrecheckService) RunPrecheckMulti(ctx context.Context, months []ClaimMonth, caseIDs []uuid.UUID) (*PrecheckReport, error) {
	if len(months) == 0 {
		return nil, ErrPeriodsRequired
	}

	merged := &PrecheckReport{Issues: []PrecheckIssue{}}
	seenCaseProfile := make(map[string]bool)

	for _, month := range months {
		report, err := s.RunPrecheck(ctx, month.Scope(caseIDs))
		if err != nil {
			return nil, err
		}
		for _, issue := range report.Issues {
			if issue.Code == "MISSING_CASE_PROFILE" {
				caseID := fmt.Sprintf("%v", issue.Details["caseId"])
				if seenCaseProfile[caseID] {
					continue
				}
				seenCaseProfile[caseID] = true
			} else {
				issue.Details = withPeriodYM(issue.Details, month.PeriodYM)
			}

			merged.Issues = append(merged.Issues, issue)
			switch issue.Severity {
			case SeverityError:
				merged.TotalErrors++
			case SeverityWarning:
				merged.TotalWarnings++
			case SeverityInfo:
				merged.TotalInfos++
			}
		}
	}

	merged.Passed = merged.TotalErrors == 0
	return merged, nil
}

// withPeriodYM 複製一份 details 並補上月份，不就地改寫來源報告的 map。
func withPeriodYM(details map[string]interface{}, periodYM string) map[string]interface{} {
	copied := make(map[string]interface{}, len(details)+1)
	for k, v := range details {
		copied[k] = v
	}
	copied["periodYm"] = periodYM
	return copied
}
