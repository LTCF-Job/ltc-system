package infra

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// 產出的位元組一律再用 excelize 讀回來驗證：f.Write 沒回錯只證明「寫得出去」，
// 證明不了「檔案打得開」，更證明不了「欄位是對的」。
func TestRenderSiteTripSummary_RoundTrip(t *testing.T) {
	siteID := uuid.New()
	report := app.SiteTripSummaryReport{
		PeriodYMs: []string{"11505", "11506"},
		Sites: []app.SiteTripSummaryGroup{{
			SiteID:   &siteID,
			SiteName: "湖口老人會",
			Region:   "新竹",
			Rows: []app.SiteTripSummaryRow{
				{CaseID: uuid.New(), CaseName: "王阿金", Counts: map[string]int{"11505": 18, "11506": 20}, Total: 38},
				{CaseID: uuid.New(), CaseName: "池文水", Counts: map[string]int{"11505": 22, "11506": 0}, Total: 22},
			},
			Subtotals:     map[string]int{"11505": 40, "11506": 20},
			SubtotalTotal: 60,
		}},
	}

	data, err := NewExcelRenderer().RenderSiteTripSummary(report)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err, "產生的據點趟數彙總表必須為合法 Excel 檔案")
	defer f.Close()

	sheets := f.GetSheetList()
	require.Len(t, sheets, 1)
	assert.Equal(t, "湖口老人會", sheets[0], "一據點一張工作表，表名即據點名稱")

	rows, err := f.GetRows(sheets[0])
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 8)

	assert.Equal(t, "長照交通接送 據點趟數彙總表", rows[0][0])
	assert.Equal(t, "據點：湖口老人會（新竹）", rows[1][0])
	assert.Equal(t, "統計月份：115-05、115-06", rows[2][0])

	// 第 5 列（索引 4）是表頭：個案姓名 + 各月份 + 跨月合計
	assert.Equal(t, []string{"個案姓名", "115-05", "115-06", "跨月合計"}, rows[4])

	assert.Equal(t, []string{"王阿金", "18", "20", "38"}, rows[5])
	// 該月零趟仍要寫出 0，不可留白——留白讀起來像「沒統計到」而不是「沒搭車」。
	assert.Equal(t, []string{"池文水", "22", "0", "22"}, rows[6])
	assert.Equal(t, []string{"據點小計", "40", "20", "60"}, rows[7])
}

func TestRenderSiteTripSummary_MultipleSitesUseSeparateSheets(t *testing.T) {
	report := app.SiteTripSummaryReport{
		PeriodYMs: []string{"11505"},
		Sites: []app.SiteTripSummaryGroup{
			{SiteName: "湖口老人會", Subtotals: map[string]int{"11505": 0}},
			{SiteName: "竹北社區", Subtotals: map[string]int{"11505": 0}},
		},
	}

	data, err := NewExcelRenderer().RenderSiteTripSummary(report)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	assert.Equal(t, []string{"湖口老人會", "竹北社區"}, f.GetSheetList())
}

func TestRenderSiteTripSummary_EmptyReportStillOpens(t *testing.T) {
	// 服務層會先擋掉「沒有任何個案」的情況，但 renderer 自己也不該產出壞檔案。
	data, err := NewExcelRenderer().RenderSiteTripSummary(app.SiteTripSummaryReport{PeriodYMs: []string{"11505"}})
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetList()[0])
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 4)
	assert.Equal(t, "所選據點底下沒有可統計的個案", rows[3][0])
}
