package infra

import (
	"archive/zip"
	"bytes"
	"io"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/domain/govform"
)

// 以下期望值一律為主管機關 .xls 範本的實測結果，刻意寫成字面值而非引用
// production 常數：若實作與範本任一方改動，測試才會變紅。
const (
	tmplFontName      = "DFKai-SB"
	tmplFontSize      = 12.0
	tmplRequiredColor = "FF0000"
	tmplNormalColor   = "000000"
	tmplHeaderHeight  = 99.0
	tmplDataHeight    = 16.5
	tmplLastRequired  = 11 // 第 1～11 欄（身分證字號 ～ 結束時段-分鐘）為紅字必填
	tmplWrapFromCol   = 3  // 範本前兩欄標題不換行，第 3 欄起才換行
)

// tmplColWidths 為範本 33 欄的欄寬實測值。
var tmplColWidths = [33]float64{
	14.62, 23.75, 14.62, 10.75, 10.12, 10.25,
	18.12, 16.87, 19.62, 16.87, 19.62, 16.87,
	18.12, 18.12, 18.12, 18.12, 15.37, 15.75,
	43.37, 19.37, 19.37, 19.37, 24.25, 17.37,
	39.62, 40.62, 18.62, 18.62, 18.62, 18.62,
	18.62, 18.75, 31.37,
}

// renderTwoRowClaim 產生兩列資料的申報檔並開啟，供版面相關測試共用。
func renderTwoRowClaim(t *testing.T) *excelize.File {
	t.Helper()

	rows := make([]govform.ClaimRow, 2)
	for i := range rows {
		rows[i].Cells[0] = "A202559750"
		rows[i].Cells[1] = "1150701"
		rows[i].Cells[2] = "BD03"
	}

	data, err := NewExcelRenderer().RenderGovClaim(rows)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func styleAt(t *testing.T, f *excelize.File, axis string) *excelize.Style {
	t.Helper()

	styleID, err := f.GetCellStyle(govform.GovClaimSheetName, axis)
	require.NoError(t, err)
	style, err := f.GetStyle(styleID)
	require.NoError(t, err)
	return style
}

func TestRenderGovClaim_HeaderMatchesOfficialTemplate(t *testing.T) {
	f := renderTwoRowClaim(t)

	t.Run("前 11 欄為紅字、其餘為黑字", func(t *testing.T) {
		for col := 1; col <= 33; col++ {
			axis, err := excelize.CoordinatesToCellName(col, 1)
			require.NoError(t, err)

			style := styleAt(t, f, axis)
			require.NotNil(t, style.Font, "第 %d 欄標題應有字型設定", col)

			want := tmplNormalColor
			if col <= tmplLastRequired {
				want = tmplRequiredColor
			}
			assert.Equal(t, want, style.Font.Color, "第 %d 欄標題顏色", col)
		}
	})

	t.Run("標題使用標楷體 12pt 並置中加框", func(t *testing.T) {
		style := styleAt(t, f, "A1")

		require.NotNil(t, style.Font)
		assert.Equal(t, tmplFontName, style.Font.Family)
		assert.Equal(t, tmplFontSize, style.Font.Size)

		require.NotNil(t, style.Alignment)
		assert.Equal(t, "center", style.Alignment.Horizontal)
		assert.Equal(t, "center", style.Alignment.Vertical)

		assert.Len(t, style.Border, 4, "標題四邊皆須有框線")
		for _, b := range style.Border {
			assert.Equal(t, 1, b.Style, "%s 邊框應為細實線", b.Type)
		}
	})

	t.Run("前兩欄標題不換行、第 3 欄起換行", func(t *testing.T) {
		for col := 1; col <= 33; col++ {
			axis, err := excelize.CoordinatesToCellName(col, 1)
			require.NoError(t, err)

			style := styleAt(t, f, axis)
			require.NotNil(t, style.Alignment, "第 %d 欄標題應有對齊設定", col)
			assert.Equal(t, col >= tmplWrapFromCol, style.Alignment.WrapText,
				"第 %d 欄標題換行設定", col)
		}
	})
}

func TestRenderGovClaim_DataRowsUseTemplateLayout(t *testing.T) {
	f := renderTwoRowClaim(t)
	style := styleAt(t, f, "A2")

	require.NotNil(t, style.Font)
	assert.Equal(t, tmplFontName, style.Font.Family)
	assert.Equal(t, tmplFontSize, style.Font.Size)
	assert.Equal(t, tmplNormalColor, style.Font.Color)

	require.NotNil(t, style.Alignment)
	assert.Equal(t, "center", style.Alignment.Horizontal)

	assert.Empty(t, style.Border, "資料列不加框線，與官方範本一致")
}

func TestRenderGovClaim_ColumnWidthsAndRowHeights(t *testing.T) {
	f := renderTwoRowClaim(t)
	sheet := govform.GovClaimSheetName

	t.Run("欄寬符合範本", func(t *testing.T) {
		for colIdx, want := range tmplColWidths {
			colName, err := excelize.ColumnNumberToName(colIdx + 1)
			require.NoError(t, err)

			got, err := f.GetColWidth(sheet, colName)
			require.NoError(t, err)
			assert.InDelta(t, want, got, 0.01, "第 %d 欄欄寬", colIdx+1)
		}
	})

	t.Run("列高符合範本", func(t *testing.T) {
		headerHigh, err := f.GetRowHeight(sheet, 1)
		require.NoError(t, err)
		assert.InDelta(t, tmplHeaderHeight, headerHigh, 0.01)

		for _, row := range []int{2, 3} {
			dataHigh, err := f.GetRowHeight(sheet, row)
			require.NoError(t, err)
			assert.InDelta(t, tmplDataHeight, dataHigh, 0.01, "第 %d 列列高", row)
		}
	})
}

// 無資料時只有標題列，不應因為套用資料樣式而失敗。
func TestRenderGovClaim_EmptyRowsStillRenders(t *testing.T) {
	data, err := NewExcelRenderer().RenderGovClaim(nil)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	styleID, err := f.GetCellStyle(govform.GovClaimSheetName, "A1")
	require.NoError(t, err)
	style, err := f.GetStyle(styleID)
	require.NoError(t, err)
	require.NotNil(t, style.Font)
	assert.Equal(t, tmplRequiredColor, style.Font.Color)
}

// 產出的 styles.xml 色碼必須是合法的 6 或 8 碼 aRGB。
//
// excelize 的 Style.Color 期望 6 碼 RGB 並自行補上 alpha；若呼叫端誤帶 alpha
// 會寫出 10 碼色碼——excelize 自己讀得回來，但 openpyxl 等外部讀取器會直接
// 拒絕整份檔案。因此這裡直接驗原始 XML，而非 excelize 的來回轉換結果。
func TestRenderGovClaim_StylesXMLUsesValidARGB(t *testing.T) {
	rows := make([]govform.ClaimRow, 1)
	rows[0].Cells[0] = "A202559750"

	data, err := NewExcelRenderer().RenderGovClaim(rows)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	var styles string
	for _, zf := range zr.File {
		if zf.Name != "xl/styles.xml" {
			continue
		}
		rc, err := zf.Open()
		require.NoError(t, err)
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		require.NoError(t, err)
		styles = string(raw)
	}
	require.NotEmpty(t, styles, "產出的檔案必須包含 xl/styles.xml")

	matches := regexp.MustCompile(`rgb="([0-9A-Fa-f]*)"`).FindAllStringSubmatch(styles, -1)
	require.NotEmpty(t, matches, "styles.xml 應含色碼設定")
	for _, m := range matches {
		length := len(m[1])
		assert.True(t, length == 6 || length == 8,
			"色碼 %q 長度為 %d，合法 aRGB 應為 6 或 8 碼", m[1], length)
	}
}
