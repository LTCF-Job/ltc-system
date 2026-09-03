package infra

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/domain/govform"
)

func TestExcelGenerators_ValidOpenXML(t *testing.T) {
	t.Run("GovClaimExcel", func(t *testing.T) {
		var cells [33]interface{}
		cells[0] = "A202559750"
		cells[1] = "張曾阿妹"
		cells[2] = "苗栗縣"
		rows := []govform.ClaimRow{
			{
				Cells: cells,
			},
		}
		data, err := NewExcelRenderer().RenderGovClaim(rows)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		f, err := excelize.OpenReader(bytes.NewReader(data))
		require.NoError(t, err, "產生的 GovClaimExcel 必須為合法 Excel 檔案")
		defer f.Close()

		sheetName := f.GetSheetName(0)
		assert.Equal(t, govform.GovClaimSheetName, sheetName)
	})

	t.Run("TripSummaryExcel", func(t *testing.T) {
		vehicles := []app.TripSummaryVehicle{
			{
				VehicleName:      "長照1號車",
				PlateNo:          "ABC-1234",
				SubtotalOutbound: 10,
				SubtotalInbound:  10,
				SubtotalTotal:    20,
				Rows: []app.TripSummaryCaseRow{
					{
						CaseName:      "王大明",
						OutboundCount: 5,
						InboundCount:  5,
						TotalCount:    10,
					},
				},
			},
		}
		data, err := NewExcelRenderer().RenderTripSummary("115-07", vehicles)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		f, err := excelize.OpenReader(bytes.NewReader(data))
		require.NoError(t, err, "產生的 TripSummaryExcel 必須為合法 Excel 檔案")
		defer f.Close()

		sheets := f.GetSheetList()
		assert.Contains(t, sheets, "長照1號車")
		fmt.Printf("TRIP_SUMMARY_BASE64=%s\n", base64.StdEncoding.EncodeToString(data))
	})

	t.Run("HsinchuScheduleExcel", func(t *testing.T) {
		outbound := []app.HsinchuScheduleItem{
			{
				Direction:   "outbound",
				RunNo:       1,
				CaseName:    "張小美",
				DepartTime:  "08:30",
				Origin:      "竹北市光明六路1號",
				Destination: "竹北日照中心",
				VehicleName: "長照2號車",
				SiteName:    "竹北日照",
			},
		}
		data, err := NewExcelRenderer().RenderHsinchuSchedule(outbound, nil)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		f, err := excelize.OpenReader(bytes.NewReader(data))
		require.NoError(t, err, "產生的 HsinchuScheduleExcel 必須為合法 Excel 檔案")
		defer f.Close()

		sheet := f.GetSheetName(0)
		assert.Equal(t, "新竹接送時刻表", sheet)
	})
}
