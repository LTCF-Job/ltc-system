package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

func TestReportService_GenerateTripSummaryExcel(t *testing.T) {
	svc := NewReportService(nil)

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
