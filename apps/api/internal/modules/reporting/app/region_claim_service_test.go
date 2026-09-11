package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// monthScopedSourceReader 只在指定月份回傳來源，用來模擬「某些月份有資料、某些沒有」。
type monthScopedSourceReader struct {
	byMonth map[string][]app.GovClaimSource
	scopes  []app.ClaimScope
}

func (r *monthScopedSourceReader) QueryGovClaimSources(_ context.Context, scope app.ClaimScope) ([]app.GovClaimSource, error) {
	r.scopes = append(r.scopes, scope)
	key := scope.StartDate.Format("2006-01")
	return r.byMonth[key], nil
}

// newRegionSource 產生一筆可通過驗證的申報來源；日期只用來區分不同筆資料。
func newRegionSource(t *testing.T, caseID uuid.UUID, day int) app.GovClaimSource {
	t.Helper()
	return newSource(t, caseID, "C001", "蔡曾切", uuid.New(), day, 1, "outbound", "09:00")
}

func TestRegionClaimService_CreateRegionClaimJobs(t *testing.T) {
	caseID := uuid.New()
	scopedCases := []app.ScopedCase{{ID: caseID, Name: "蔡曾切", Region: "新竹"}}

	newRegionService := func(reader app.GovClaimSourceReader, resolver app.ClaimCaseResolver) *app.RegionClaimService {
		store := &fakeExportStore{jobID: uuid.New()}
		claims := newService(reader, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{})
		return app.NewRegionClaimService(resolver, claims)
	}

	t.Run("逐月各建立一個匯出工作", func(t *testing.T) {
		// 一月一 job 是硬性限制：export_job_files 有 UNIQUE(job_id, case_id)，
		// 同一位個案的不同月份塞不進同一個 job。
		reader := &monthScopedSourceReader{byMonth: map[string][]app.GovClaimSource{
			"2026-05": {newRegionSource(t, caseID, 1)},
			"2026-06": {newRegionSource(t, caseID, 2)},
		}}
		svc := newRegionService(reader, fakeClaimCaseResolver{cases: scopedCases})

		result, err := svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{
			Regions:   []string{"新竹"},
			PeriodYMs: []string{"11506", "11505"},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, result.CaseCount)
		assert.Equal(t, []string{"11505", "11506"}, result.PeriodYMs, "月份需去重並升冪排序")
		require.Len(t, result.Months, 2)
		assert.True(t, result.Months[0].Succeeded)
		assert.True(t, result.Months[1].Succeeded)
		assert.Len(t, result.SucceededJobIDs, 2)
		require.Len(t, reader.scopes, 2, "每個月份各查一次來源")
	})

	t.Run("單一月份查無資料不影響其餘月份", func(t *testing.T) {
		// 使用者要的是先拿到報得出來的月份，不是被整批擋下。
		reader := &monthScopedSourceReader{byMonth: map[string][]app.GovClaimSource{
			"2026-05": {newRegionSource(t, caseID, 1)},
		}}
		svc := newRegionService(reader, fakeClaimCaseResolver{cases: scopedCases})

		result, err := svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{
			Regions:   []string{"新竹"},
			PeriodYMs: []string{"11505", "11506"},
		})

		require.NoError(t, err)
		require.Len(t, result.Months, 2)
		assert.True(t, result.Months[0].Succeeded)
		assert.False(t, result.Months[1].Succeeded)
		assert.Equal(t, "該月份沒有可申報的資料", result.Months[1].ErrorMessage)
		assert.Len(t, result.SucceededJobIDs, 1)
	})

	t.Run("所有月份都沒有資料時整批回 ErrNoExportData", func(t *testing.T) {
		svc := newRegionService(&monthScopedSourceReader{}, fakeClaimCaseResolver{cases: scopedCases})

		_, err := svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{
			Regions:   []string{"新竹"},
			PeriodYMs: []string{"11505", "11506"},
		})

		assert.ErrorIs(t, err, app.ErrNoExportData)
	})

	t.Run("區域底下沒有個案時不建立任何工作", func(t *testing.T) {
		// 條件下錯就擋在建立工作之前，不留一串必然失敗的歷史紀錄。
		reader := &monthScopedSourceReader{}
		svc := newRegionService(reader, fakeClaimCaseResolver{cases: nil})

		_, err := svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{
			Regions:   []string{"不存在的區域"},
			PeriodYMs: []string{"11505"},
		})

		assert.ErrorIs(t, err, app.ErrNoExportData)
		assert.Empty(t, reader.scopes, "個案都解析不到就不該再去查申報來源")
	})

	t.Run("所有月份都因未裁決混車衝突被擋下時回 ErrPrecheckBlocked", func(t *testing.T) {
		// 「有資料但被擋」跟「沒有資料」是兩回事：全部月份都卡在混車衝突未裁決時，
		// 應提示使用者去裁決，而不是誤導成條件下沒有可申報資料。
		reader := &monthScopedSourceReader{byMonth: map[string][]app.GovClaimSource{
			"2026-05": {newRegionSource(t, caseID, 1)},
			"2026-06": {newRegionSource(t, caseID, 2)},
		}}
		store := &fakeExportStore{jobID: uuid.New()}
		precheck := stubPrecheckRepo{conflicts: []app.UnresolvedConflict{
			{RideID: uuid.New(), CaseName: "蔡曾切", ServiceDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		}}
		claims := newService(reader, store, &recordingRenderer{}, &recordingArchiver{}, precheck)
		svc := app.NewRegionClaimService(fakeClaimCaseResolver{cases: scopedCases}, claims)

		_, err := svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{
			Regions:   []string{"新竹"},
			PeriodYMs: []string{"11505", "11506"},
		})

		assert.ErrorIs(t, err, app.ErrPrecheckBlocked)
	})

	t.Run("所有月份都因內部錯誤失敗時不偽裝成沒有資料", func(t *testing.T) {
		reader := &monthScopedSourceReader{byMonth: map[string][]app.GovClaimSource{
			"2026-05": {newRegionSource(t, caseID, 1)},
			"2026-06": {newRegionSource(t, caseID, 2)},
		}}
		store := &fakeExportStore{jobID: uuid.New(), createFail: errors.New("db connection lost")}
		claims := newService(reader, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{})
		svc := app.NewRegionClaimService(fakeClaimCaseResolver{cases: scopedCases}, claims)

		_, err := svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{
			Regions:   []string{"新竹"},
			PeriodYMs: []string{"11505", "11506"},
		})

		require.Error(t, err)
		assert.NotErrorIs(t, err, app.ErrNoExportData, "含有內部錯誤時不該說成查無資料")
		assert.NotErrorIs(t, err, app.ErrPrecheckBlocked)
	})

	t.Run("未指定區域或月份時拒絕執行", func(t *testing.T) {
		svc := newRegionService(&monthScopedSourceReader{}, fakeClaimCaseResolver{cases: scopedCases})

		_, err := svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{PeriodYMs: []string{"11505"}})
		assert.ErrorIs(t, err, app.ErrRegionsRequired)

		_, err = svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{Regions: []string{"新竹"}})
		assert.ErrorIs(t, err, app.ErrPeriodsRequired)
	})

	t.Run("批次模式固定產壓縮檔並記下範圍來源", func(t *testing.T) {
		reader := &monthScopedSourceReader{byMonth: map[string][]app.GovClaimSource{
			"2026-05": {newRegionSource(t, caseID, 1)},
		}}
		store := &fakeExportStore{jobID: uuid.New()}
		claims := newService(reader, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{})
		svc := app.NewRegionClaimService(fakeClaimCaseResolver{cases: scopedCases}, claims)

		_, err := svc.CreateRegionClaimJobs(context.Background(), app.RegionClaimInput{
			Regions:   []string{"新竹"},
			PeriodYMs: []string{"11505"},
		})

		require.NoError(t, err)
		require.Len(t, store.created, 1)
		assert.Equal(t, "zip", store.created[0].Format, "一次數十份檔案沒有逐案點下載的道理")
		assert.Equal(t, []uuid.UUID{caseID}, store.created[0].CaseIDs, "解析結果要固化進工作，歷史才查得出當時報了誰")
	})
}

func TestBatchZipFileName(t *testing.T) {
	assert.Equal(t, "gov-claim-11505-11507.zip", app.BatchZipFileName([]string{"11505", "11506", "11507"}))
	assert.Equal(t, "gov-claim-11505.zip", app.BatchZipFileName([]string{"11505"}), "單月退回既有檔名格式")
	assert.Equal(t, "gov-claim.zip", app.BatchZipFileName(nil))
}
