package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWeekdays(t *testing.T) {
	tests := []struct {
		input string
		want  []int16
	}{
		{"週一到週五", []int16{1, 2, 3, 4, 5}},
		{"周一到周五", []int16{1, 2, 3, 4, 5}},
		{"週一~週五", []int16{1, 2, 3, 4, 5}},
		{"週四、週五", []int16{4, 5}},
		{"週二，週四下午去回", []int16{2, 4}},
		{"周一早上來回", []int16{1}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseWeekdays(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseCasesFromExcel_RealFile(t *testing.T) {
	filePath := filepath.Join("..", "..", "..", "..", "source", "個案新增資料.xlsx")
	f, err := os.Open(filePath)
	if err != nil {
		t.Skip("Sample file not found, skipping real file test")
		return
	}
	defer f.Close()

	svc := NewImportService(nil, nil, nil, nil, nil)
	preview, err := svc.ParseCasesFromExcel(f)
	require.NoError(t, err)
	require.NotNil(t, preview)

	assert.Greater(t, preview.TotalRows, 0)
	t.Logf("Parsed %d case rows, %d valid, %d with missing required fields",
		preview.TotalRows, preview.ValidRows, preview.ErrorRows)
}

func TestParseScheduleWorkbook_RealFile(t *testing.T) {
	filePath := filepath.Join("..", "..", "..", "..", "source", "(參考用)交通車接送班表.xlsx")
	f, err := os.Open(filePath)
	if err != nil {
		t.Skip("Sample file not found, skipping real file test")
		return
	}
	defer f.Close()

	svc := NewImportService(nil, nil, nil, nil, nil)
	sites, drivers, err := svc.ParseScheduleWorkbook(f)
	require.NoError(t, err)

	assert.NotEmpty(t, sites)
	assert.NotEmpty(t, drivers)
	t.Logf("Extracted %d sites and %d drivers from schedule workbook", len(sites), len(drivers))
}
