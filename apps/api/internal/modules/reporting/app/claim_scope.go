package app

import (
	"sort"
	"time"

	"github.com/google/uuid"
)

// ClaimScope 是申報前置檢核與申報來源查詢共用的資料範圍。
type ClaimScope struct {
	StartDate time.Time
	EndDate   time.Time
	CaseIDs   []uuid.UUID
}

// NewClaimScope 建立申報範圍。
func NewClaimScope(startDate, endDate time.Time, caseIDs []uuid.UUID) ClaimScope {
	return ClaimScope{
		StartDate: startDate,
		EndDate:   endDate,
		CaseIDs:   append([]uuid.UUID(nil), caseIDs...),
	}
}

// ClaimMonth 是一個民國月份與其對應的西元起訖（左閉右開）。
// 多月匯出一律逐月展開成多個 ClaimScope，不合併成 min~max 單一區間：
// 使用者選的月份可能不連續，而且申報作業本來就以月為單位。
type ClaimMonth struct {
	PeriodYM  string
	StartDate time.Time
	EndDate   time.Time
}

// Scope 以這個月份與指定個案組出申報範圍。
func (m ClaimMonth) Scope(caseIDs []uuid.UUID) ClaimScope {
	return NewClaimScope(m.StartDate, m.EndDate, caseIDs)
}

// ParseClaimMonths 解析民國 5 碼月份清單，去重後依月份升冪排序。
// 任一筆格式錯誤即整批拒絕，避免只匯出一半月份卻沒人察覺。
func ParseClaimMonths(raw []string) ([]ClaimMonth, error) {
	if len(raw) == 0 {
		return nil, ErrPeriodsRequired
	}

	seen := make(map[string]bool, len(raw))
	months := make([]ClaimMonth, 0, len(raw))
	for _, item := range raw {
		periodYM, start, end, err := ParseClaimPeriod(item)
		if err != nil {
			return nil, err
		}
		if seen[periodYM] {
			continue
		}
		seen[periodYM] = true
		months = append(months, ClaimMonth{PeriodYM: periodYM, StartDate: start, EndDate: end})
	}

	sort.Slice(months, func(i, j int) bool { return months[i].PeriodYM < months[j].PeriodYM })
	return months, nil
}
