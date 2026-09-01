package infra_test

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/modules/reporting/infra"
)

// TestZipArchiver_EntriesRemainOpenableWorkbooks 打包後每個項目仍須能被試算表解析器開啟。
// 只確認 BuildZip 沒回傳錯誤並不能證明壓縮檔裡的工作簿沒被破壞。
func TestZipArchiver_EntriesRemainOpenableWorkbooks(t *testing.T) {
	renderer := infra.NewExcelRenderer()

	first, err := renderer.RenderGovClaim(sampleClaimRows(t, "A202559750"))
	require.NoError(t, err)
	second, err := renderer.RenderGovClaim(sampleClaimRows(t, "B123456789"))
	require.NoError(t, err)

	archive, err := infra.NewZipArchiver().BuildZip([]app.ZipEntry{
		{Name: "蔡曾切11507.xlsx", Content: first},
		{Name: "林大明11507.xlsx", Content: second},
	})
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	require.NoError(t, err)
	require.Len(t, reader.File, 2)

	wantNames := []string{"蔡曾切11507.xlsx", "林大明11507.xlsx"}
	for i, entry := range reader.File {
		assert.Equal(t, wantNames[i], entry.Name)

		opened, err := entry.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(opened)
		require.NoError(t, err)

		f, err := excelize.OpenReader(bytes.NewReader(content))
		require.NoError(t, err)
		rows, err := f.GetRows(govform.GovClaimSheetName)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		require.Len(t, rows, 2, "應有 1 列標題與 1 列資料")
		assert.Equal(t, govform.Headers33[:], rows[0])
	}
}

func TestZipArchiver_EmptyArchiveIsStillValid(t *testing.T) {
	archive, err := infra.NewZipArchiver().BuildZip(nil)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	require.NoError(t, err)
	assert.Empty(t, reader.File)
}

func sampleClaimRows(t *testing.T, nationalID string) []govform.ClaimRow {
	t.Helper()
	row, err := govform.BuildClaimRow(govform.ClaimRowInput{
		NationalIDPlain:  nationalID,
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
	return []govform.ClaimRow{row}
}
