package infra

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// TestExportLineRow_SnapshotSurvivesJSONRoundTrip 驗證申報列存進 export_lines.raw_payload 再讀回、
// 重繪出來的工作簿內容與當初產生時完全相同。JSON 沒有整數型別，所有數值讀回都會變成 float64，
// 若沒有這一步，下載歷史檔案時的儲存格內容可能與產生當下不一致。
func TestExportLineRow_SnapshotSurvivesJSONRoundTrip(t *testing.T) {
	row, err := govform.BuildClaimRow(govform.ClaimRowInput{
		NationalIDPlain:  "A202559750",
		ServiceDate:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		ServiceCode:      "BD03",
		ServiceCategory:  1,
		UnitPrice:        115,
		DriverNationalID: "K120098177",
		DepartTime:       time.Date(2026, 7, 1, 9, 40, 0, 0, time.UTC),
		DurationMin:      10,
		Direction:        "outbound",
		LegSeq:           1,
		HomeAddress:      "新竹縣竹北市光明六路264號",
		SiteAddress:      "新竹縣竹北市中正西路100號",
		DistanceKM:       5,
		PlateNo:          "BZG-7915",
		ServiceUsageType: 2,
	})
	require.NoError(t, err)

	renderer := NewExcelRenderer()
	original, err := renderer.RenderGovClaim([]govform.ClaimRow{row})
	require.NoError(t, err)

	// 比照 service 落地時的處理：兩個身分證欄位留空，只保留 driverId
	payload := app.ClaimLinePayload{Cells: row.Cells, Direction: row.Direction, LegSeq: row.LegSeq}
	payload.Cells[0] = ""
	payload.Cells[6] = ""
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "A202559750", "個案身分證明文不得進入 raw_payload")
	assert.NotContains(t, string(encoded), "K120098177", "服務人員身分證明文不得進入 raw_payload")

	restored, err := exportLineRow{LineNo: 1, ServiceDateROC: 1150701, RawPayload: encoded}.toApp()
	require.NoError(t, err)

	rebuiltRow := govform.ClaimRow{
		Cells:     restored.Payload.Cells,
		Direction: restored.Payload.Direction,
		LegSeq:    restored.Payload.LegSeq,
	}
	rebuiltRow.Cells[0] = "A202559750"
	rebuiltRow.Cells[6] = "K120098177"

	rebuilt, err := renderer.RenderGovClaim([]govform.ClaimRow{rebuiltRow})
	require.NoError(t, err)

	assert.Equal(t, readClaimSheet(t, original), readClaimSheet(t, rebuilt))
}

func readClaimSheet(t *testing.T, content []byte) [][]string {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(content))
	require.NoError(t, err)
	defer f.Close()

	rows, err := f.GetRows(govform.GovClaimSheetName)
	require.NoError(t, err)
	return rows
}
