package app_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/modules/reporting/infra"
)

type emptyReportRepository struct{}

func (emptyReportRepository) QueryTripSummaryData(context.Context, time.Time, time.Time, *string, *uuid.UUID) ([]app.ReportVehicleTripSummary, error) {
	return []app.ReportVehicleTripSummary{}, nil
}

func (emptyReportRepository) QueryHsinchuScheduleData(context.Context, *uuid.UUID, *uuid.UUID) ([]app.ReportHsinchuScheduleRow, error) {
	return []app.ReportHsinchuScheduleRow{}, nil
}

func TestReportService_GenerateTripSummaryExcel(t *testing.T) {
	svc := app.NewReportService(emptyReportRepository{}, infra.NewExcelRenderer())

	ctx := context.Background()
	excelBytes, err := svc.GenerateTripSummaryExcel(ctx, "115-07", nil, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, excelBytes)

	// 驗證能否以 excelize 開啟
	f, err := excelize.OpenReader(bytes.NewReader(excelBytes))
	assert.NoError(t, err)
	defer f.Close()

	sheetList := f.GetSheetList()
	assert.NotEmpty(t, sheetList)
}

func TestReportService_GenerateHsinchuScheduleExcel(t *testing.T) {
	svc := app.NewReportService(emptyReportRepository{}, infra.NewExcelRenderer())

	ctx := context.Background()
	excelBytes, err := svc.GenerateHsinchuScheduleExcel(ctx, nil, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, excelBytes)

	f, err := excelize.OpenReader(bytes.NewReader(excelBytes))
	assert.NoError(t, err)
	defer f.Close()

	sheetList := f.GetSheetList()
	assert.Contains(t, sheetList, "新竹接送時刻表")
}
