package export

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestVerifyCleanOpenXMLOutput(t *testing.T) {
	vehicles := []TripSummaryExportVehicle{
		{
			VehicleName:      "長照1號車",
			PlateNo:          "ABC-1234",
			SubtotalOutbound: 10,
			SubtotalInbound:  10,
			SubtotalTotal:    20,
			Rows: []TripSummaryExportCaseRow{
				{
					CaseCode:      "C001",
					CaseName:      "王大明",
					OutboundCount: 5,
					InboundCount:  5,
					TotalCount:    10,
				},
			},
		},
	}

	data, err := GenerateTripSummaryExcel("115-07", vehicles)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	dimension, err := f.GetSheetDimension("長照1號車")
	require.NoError(t, err)
	fmt.Printf("Sheet Dimension: %s\n", dimension)
	fmt.Printf("NEW_BASE64: %s\n", base64.StdEncoding.EncodeToString(data))
}
