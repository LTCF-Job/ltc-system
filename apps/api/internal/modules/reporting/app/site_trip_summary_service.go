package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// SiteTripSummaryReport 是據點趟數彙總表的完整內容：一個據點一組，
// 每位個案一列，每個選取月份一欄，最後加上跨月合計。
type SiteTripSummaryReport struct {
	PeriodYMs []string
	Sites     []SiteTripSummaryGroup
}

// SiteTripSummaryGroup 代表單一據點的趟數統計。
type SiteTripSummaryGroup struct {
	SiteID   *uuid.UUID
	SiteName string
	Region   string
	Rows     []SiteTripSummaryRow
	// Subtotals 是該據點各月份的小計，key 為民國 5 碼。
	Subtotals map[string]int
	// SubtotalTotal 是該據點跨月的總趟數。
	SubtotalTotal int
}

// SiteTripSummaryRow 代表單一個案在各月份的趟數。
type SiteTripSummaryRow struct {
	CaseID   uuid.UUID
	CaseName string
	// Counts 以民國 5 碼為 key；每個選取月份都保證有值，沒有搭乘就是 0。
	Counts map[string]int
	// Total 是這位個案跨越所有選取月份的加總。
	Total int
}

// SiteTripSummaryService 產生據點趟數彙總表。
//
// 這份檔案不是政府申報檔，因此刻意不寫進 export_jobs／export_lines／export_job_files：
// 那三張表的語意是「這次報給政府的東西的不可變快照」，混進一份管理用統計表會污染申報
// 稽核軌跡，也用不到逐案 caseId 的下載路徑。改為同步產檔直接回傳位元組，比照
// GET /reports/trip-summary/export 的既有作法。
type SiteTripSummaryService struct {
	scope    ClaimCaseResolver
	reader   SiteTripSummaryReader
	renderer SiteTripSummaryRenderer
}

// NewSiteTripSummaryService 建立 SiteTripSummaryService 實例。
func NewSiteTripSummaryService(
	scope ClaimCaseResolver,
	reader SiteTripSummaryReader,
	renderer SiteTripSummaryRenderer,
) *SiteTripSummaryService {
	return &SiteTripSummaryService{scope: scope, reader: reader, renderer: renderer}
}

// Generate 產生據點趟數彙總表，回傳檔名與工作簿位元組。
func (s *SiteTripSummaryService) Generate(
	ctx context.Context,
	siteIDs []uuid.UUID,
	periodYMs []string,
) (string, []byte, error) {
	if len(siteIDs) == 0 {
		return "", nil, ErrSiteIDsRequired
	}
	months, err := ParseClaimMonths(periodYMs)
	if err != nil {
		return "", nil, err
	}

	cases, err := s.scope.ListCasesBySites(ctx, siteIDs)
	if err != nil {
		return "", nil, fmt.Errorf("list cases by sites: %w", err)
	}
	// 「這幾個據點底下一位個案都沒有」是條件下錯，要明講。
	// 但「有個案、只是這幾個月都沒跑」不是錯誤——全 0 的表本身就是使用者要的答案，
	// 照樣產出。這條分界不要混為一談。
	if len(cases) == 0 {
		return "", nil, ErrNoExportData
	}

	counts, err := s.reader.QuerySiteTripCounts(ctx, siteIDs, months)
	if err != nil {
		return "", nil, fmt.Errorf("query site trip counts: %w", err)
	}

	report := buildSiteTripSummaryReport(cases, counts, months)
	content, err := s.renderer.RenderSiteTripSummary(report)
	if err != nil {
		return "", nil, fmt.Errorf("render site trip summary: %w", err)
	}
	return SiteTripSummaryFileName(report.PeriodYMs), content, nil
}

// SiteTripSummaryFileName 以起訖月份組出檔名；單月時不重複輸出同一個月份。
func SiteTripSummaryFileName(periodYMs []string) string {
	if len(periodYMs) == 0 {
		return "site-trip-summary.xlsx"
	}
	first := periodYMs[0]
	last := periodYMs[len(periodYMs)-1]
	if first == last {
		return fmt.Sprintf("site-trip-summary-%s.xlsx", first)
	}
	return fmt.Sprintf("site-trip-summary-%s-%s.xlsx", first, last)
}

// buildSiteTripSummaryReport 以個案清單為骨架、趟數為填充值組出報表。
//
// 骨架來自個案清單而不是趟數查詢結果：即使某位個案在所有選取月份都沒有搭乘紀錄，
// 他仍必須出現在表上（全 0），否則使用者會以為這個人不屬於這個據點。
func buildSiteTripSummaryReport(
	cases []ScopedCase,
	counts []SiteTripCount,
	months []ClaimMonth,
) SiteTripSummaryReport {
	periodYMs := make([]string, len(months))
	for i, m := range months {
		periodYMs[i] = m.PeriodYM
	}

	byCase := make(map[uuid.UUID]map[string]int, len(cases))
	for _, c := range counts {
		if byCase[c.CaseID] == nil {
			byCase[c.CaseID] = make(map[string]int, len(months))
		}
		byCase[c.CaseID][c.PeriodYM] = c.TripCount
	}

	groupIndex := make(map[string]int)
	groups := make([]SiteTripSummaryGroup, 0)

	for _, item := range cases {
		key := item.SiteName
		if item.SiteID != nil {
			key = item.SiteID.String()
		}
		pos, ok := groupIndex[key]
		if !ok {
			groups = append(groups, SiteTripSummaryGroup{
				SiteID:    item.SiteID,
				SiteName:  item.SiteName,
				Region:    item.Region,
				Subtotals: make(map[string]int, len(months)),
			})
			pos = len(groups) - 1
			groupIndex[key] = pos
		}

		row := SiteTripSummaryRow{
			CaseID:   item.ID,
			CaseName: item.Name,
			Counts:   make(map[string]int, len(months)),
		}
		for _, periodYM := range periodYMs {
			count := byCase[item.ID][periodYM]
			row.Counts[periodYM] = count
			row.Total += count
			groups[pos].Subtotals[periodYM] += count
			groups[pos].SubtotalTotal += count
		}
		groups[pos].Rows = append(groups[pos].Rows, row)
	}

	sort.SliceStable(groups, func(i, j int) bool { return groups[i].SiteName < groups[j].SiteName })
	return SiteTripSummaryReport{PeriodYMs: periodYMs, Sites: groups}
}
