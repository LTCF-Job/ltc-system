package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	Passed       bool            `json:"passed"` // 無 error 時為 true
	TotalErrors  int             `json:"totalErrors"`
	TotalWarnings int            `json:"totalWarnings"`
	TotalInfos   int             `json:"totalInfos"`
	Issues       []PrecheckIssue `json:"issues"`
}

// PrecheckService 負責在匯出前執行全方位資料驗證。
type PrecheckService struct {
	db *pgxpool.Pool
}

// NewPrecheckService 建立 PrecheckService 實例。
func NewPrecheckService(db *pgxpool.Pool) *PrecheckService {
	return &PrecheckService{db: db}
}

// RunPrecheck 執行指定月份與區域之申報前置檢核（規格書 7.6）。
func (s *PrecheckService) RunPrecheck(ctx context.Context, periodYM string, region string) (*PrecheckReport, error) {
	var issues []PrecheckIssue
	errorCount := 0
	warningCount := 0
	infoCount := 0

	// 1. 固定 Info: 配給額度檢查未執行（Q7 決議）
	issues = append(issues, PrecheckIssue{
		Severity: SeverityInfo,
		Code:     "QUOTA_CHECK_SKIPPED",
		Message:  "個案配給額度檢查未執行——尚未取得額度計算規則",
	})
	infoCount++

	if s.db != nil {
		// 2. 檢查是否有個案缺必要欄位 (身分證、住址、使用類型)
		queryMissingCaseFields := `
			SELECT id, name
			FROM cases
			WHERE region = $1 AND status = 'active'
			  AND (home_address = '' OR service_usage_type IS NULL OR national_id_masked = '')
		`
		rows, err := s.db.Query(ctx, queryMissingCaseFields, region)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cid uuid.UUID
				var cname string
				if err := rows.Scan(&cid, &cname); err == nil {
					issues = append(issues, PrecheckIssue{
						Severity: SeverityError,
						Code:     "MISSING_CASE_PROFILE",
						Message:  fmt.Sprintf("個案「%s」缺少身分證、住家地址或服務使用類型", cname),
						Details: map[string]interface{}{
							"caseId":   cid,
							"caseName": cname,
						},
					})
					errorCount++
				}
			}
		}

		// 3. 檢查是否有未處理的混車衝突
		queryConflicts := `
			SELECT r.id, c.name, r.service_date
			FROM ride_records r
			JOIN cases c ON r.case_id = c.id
			WHERE c.region = $1 AND r.has_conflict = true AND r.conflict_resolved_at IS NULL
		`
		confRows, err := s.db.Query(ctx, queryConflicts, region)
		if err == nil {
			defer confRows.Close()
			for confRows.Next() {
				var rid uuid.UUID
				var cname string
				var sdate time.Time
				if err := confRows.Scan(&rid, &cname, &sdate); err == nil {
					issues = append(issues, PrecheckIssue{
						Severity: SeverityWarning,
						Code:     "UNRESOLVED_CONFLICT",
						Message:  fmt.Sprintf("個案「%s」於 %s 存在未裁決之混車衝突", cname, sdate.Format("2006-01-02")),
						Details: map[string]interface{}{
							"rideId":      rid,
							"caseName":    cname,
							"serviceDate": sdate.Format("2006-01-02"),
						},
					})
					warningCount++
				}
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
