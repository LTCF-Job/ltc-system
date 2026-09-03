package infra

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestVerifyCleanOpenXMLOutput(t *testing.T) {
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

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	dimension, err := f.GetSheetDimension("長照1號車")
	require.NoError(t, err)
	fmt.Printf("Sheet Dimension: %s\n", dimension)
	fmt.Printf("NEW_BASE64: %s\n", base64.StdEncoding.EncodeToString(data))
}
