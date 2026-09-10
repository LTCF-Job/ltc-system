package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

type fakeClaimCaseResolver struct {
	cases []app.ScopedCase
	err   error
}

func (f fakeClaimCaseResolver) ListCasesBySites(context.Context, []uuid.UUID) ([]app.ScopedCase, error) {
	return f.cases, f.err
}

func (f fakeClaimCaseResolver) ListCasesByRegions(context.Context, []string) ([]app.ScopedCase, error) {
	return f.cases, f.err
}

type fakeSiteTripReader struct {
	counts []app.SiteTripCount
	err    error
}

func (f fakeSiteTripReader) QuerySiteTripCounts(context.Context, []uuid.UUID, []app.ClaimMonth) ([]app.SiteTripCount, error) {
	return f.counts, f.err
}

// capturingSummaryRenderer 攔下報表結構，讓測試檢查服務組出來的內容而非 Excel 位元組。
type capturingSummaryRenderer struct {
	report app.SiteTripSummaryReport
	err    error
}

func (r *capturingSummaryRenderer) RenderSiteTripSummary(report app.SiteTripSummaryReport) ([]byte, error) {
	r.report = report
	if r.err != nil {
		return nil, r.err
	}
	return []byte("workbook"), nil
}

func TestSiteTripSummaryService_Generate(t *testing.T) {
	siteID := uuid.New()
	caseA := uuid.New()
	caseB := uuid.New()
	siteIDs := []uuid.UUID{siteID}
	cases := []app.ScopedCase{
		{ID: caseA, Name: "王阿金", SiteID: &siteID, SiteName: "湖口老人會", Region: "新竹"},
		{ID: caseB, Name: "池文水", SiteID: &siteID, SiteName: "湖口老人會", Region: "新竹"},
	}

	t.Run("逐月趟數與跨月加總", func(t *testing.T) {
		renderer := &capturingSummaryRenderer{}
		svc := app.NewSiteTripSummaryService(
			fakeClaimCaseResolver{cases: cases},
			fakeSiteTripReader{counts: []app.SiteTripCount{
				{CaseID: caseA, PeriodYM: "11505", TripCount: 18},
				{CaseID: caseA, PeriodYM: "11506", TripCount: 20},
				{CaseID: caseB, PeriodYM: "11505", TripCount: 22},
				{CaseID: caseB, PeriodYM: "11506", TripCount: 19},
			}},
			renderer,
		)

		fileName, content, err := svc.Generate(context.Background(), siteIDs, []string{"11506", "11505"})

		require.NoError(t, err)
		assert.Equal(t, "site-trip-summary-11505-11506.xlsx", fileName)
		assert.Equal(t, []byte("workbook"), content)

		require.Len(t, renderer.report.Sites, 1)
		site := renderer.report.Sites[0]
		assert.Equal(t, []string{"11505", "11506"}, renderer.report.PeriodYMs, "月份必須升冪排序")
		require.Len(t, site.Rows, 2)

		assert.Equal(t, "王阿金", site.Rows[0].CaseName)
		assert.Equal(t, 38, site.Rows[0].Total, "18 + 20")
		assert.Equal(t, "池文水", site.Rows[1].CaseName)
		assert.Equal(t, 41, site.Rows[1].Total, "22 + 19")

		assert.Equal(t, 40, site.Subtotals["11505"])
		assert.Equal(t, 39, site.Subtotals["11506"])
		assert.Equal(t, 79, site.SubtotalTotal)
	})

	t.Run("某月零趟的個案仍要出現且為零", func(t *testing.T) {
		// 使用者要的是「這個據點的每一位個案分別跑了幾趟」，零趟本身就是答案的一部分；
		// 整列消失會讓人以為這個人不屬於這個據點。
		renderer := &capturingSummaryRenderer{}
		svc := app.NewSiteTripSummaryService(
			fakeClaimCaseResolver{cases: cases},
			fakeSiteTripReader{counts: []app.SiteTripCount{
				{CaseID: caseA, PeriodYM: "11505", TripCount: 18},
			}},
			renderer,
		)

		_, _, err := svc.Generate(context.Background(), siteIDs, []string{"11505", "11506"})

		require.NoError(t, err)
		site := renderer.report.Sites[0]
		require.Len(t, site.Rows, 2)
		assert.Equal(t, 0, site.Rows[0].Counts["11506"])
		assert.Equal(t, 18, site.Rows[0].Total)

		require.Equal(t, "池文水", site.Rows[1].CaseName)
		assert.Equal(t, 0, site.Rows[1].Counts["11505"])
		assert.Equal(t, 0, site.Rows[1].Counts["11506"])
		assert.Equal(t, 0, site.Rows[1].Total)
	})

	t.Run("有個案但整批零趟仍照樣產表", func(t *testing.T) {
		// 「這個據點這幾個月都沒跑」是使用者要看的答案，不是條件下錯。
		renderer := &capturingSummaryRenderer{}
		svc := app.NewSiteTripSummaryService(
			fakeClaimCaseResolver{cases: cases},
			fakeSiteTripReader{counts: nil},
			renderer,
		)

		_, content, err := svc.Generate(context.Background(), siteIDs, []string{"11505"})

		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Equal(t, 0, renderer.report.Sites[0].SubtotalTotal)
	})

	t.Run("據點底下沒有任何個案時回 ErrNoExportData", func(t *testing.T) {
		// 對比上一則：沒有個案是條件下錯，要明講，不能產一張空表讓使用者誤以為沒問題。
		svc := app.NewSiteTripSummaryService(
			fakeClaimCaseResolver{cases: nil},
			fakeSiteTripReader{},
			&capturingSummaryRenderer{},
		)

		_, _, err := svc.Generate(context.Background(), siteIDs, []string{"11505"})

		assert.ErrorIs(t, err, app.ErrNoExportData)
	})

	t.Run("未指定據點或月份時拒絕執行", func(t *testing.T) {
		svc := app.NewSiteTripSummaryService(
			fakeClaimCaseResolver{cases: cases},
			fakeSiteTripReader{},
			&capturingSummaryRenderer{},
		)

		_, _, err := svc.Generate(context.Background(), nil, []string{"11505"})
		assert.ErrorIs(t, err, app.ErrSiteIDsRequired)

		_, _, err = svc.Generate(context.Background(), siteIDs, nil)
		assert.ErrorIs(t, err, app.ErrPeriodsRequired)
	})

	t.Run("解析個案失敗時向上回報", func(t *testing.T) {
		wantErr := errors.New("database unavailable")
		svc := app.NewSiteTripSummaryService(
			fakeClaimCaseResolver{err: wantErr},
			fakeSiteTripReader{},
			&capturingSummaryRenderer{},
		)

		_, _, err := svc.Generate(context.Background(), siteIDs, []string{"11505"})
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("單月檔名不重複輸出同一個月份", func(t *testing.T) {
		assert.Equal(t, "site-trip-summary-11505.xlsx", app.SiteTripSummaryFileName([]string{"11505"}))
	})
}

func TestSiteTripSummaryService_GroupsBySite(t *testing.T) {
	siteA := uuid.New()
	siteB := uuid.New()
	caseA := uuid.New()
	caseB := uuid.New()

	renderer := &capturingSummaryRenderer{}
	svc := app.NewSiteTripSummaryService(
		fakeClaimCaseResolver{cases: []app.ScopedCase{
			{ID: caseA, Name: "王阿金", SiteID: &siteA, SiteName: "湖口老人會"},
			{ID: caseB, Name: "池文水", SiteID: &siteB, SiteName: "竹北社區"},
		}},
		fakeSiteTripReader{counts: []app.SiteTripCount{
			{CaseID: caseA, PeriodYM: "11505", TripCount: 3},
			{CaseID: caseB, PeriodYM: "11505", TripCount: 5},
		}},
		renderer,
	)

	_, _, err := svc.Generate(context.Background(), []uuid.UUID{siteA, siteB}, []string{"11505"})

	require.NoError(t, err)
	// 據點依名稱排序讓輸出順序穩定。比較是 Go 的字串（UTF-8 位元組）順序而非中文筆畫或
	// 注音，所以「湖」(U+6E56) 排在「竹」(U+7AF9) 前面——與既有 groupByCase 對個案姓名的
	// 排序是同一套規則，重點在可重現，不在語言學上的正確。
	require.Len(t, renderer.report.Sites, 2)
	assert.Equal(t, "湖口老人會", renderer.report.Sites[0].SiteName)
	assert.Equal(t, 3, renderer.report.Sites[0].SubtotalTotal)
	assert.Equal(t, "竹北社區", renderer.report.Sites[1].SiteName)
	assert.Equal(t, 5, renderer.report.Sites[1].SubtotalTotal)
}
