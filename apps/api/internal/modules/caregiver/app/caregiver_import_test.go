package app

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// testExcelReader 是 SpreadsheetReader 的最小測試替身，直接以 excelize 讀取儲存格文字；
// 不可直接借用 caregiver/infra 的 ExcelAdapter，因為該套件同時實作 CaregiverRepository
// 而回頭匯入 app 套件，從 app 的測試檔匯入會形成匯入循環。
type testExcelReader struct{}

func (testExcelReader) ReadTables(data []byte) ([][][]string, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var tables [][][]string
	var sheetNames []string
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err == nil && len(rows) > 0 {
			tables = append(tables, rows)
			sheetNames = append(sheetNames, sheet)
		}
	}
	return tables, sheetNames, nil
}

// fakeCaregiverSiteLookup resolves a fixed set of names to sites; anything else is "not found".
type fakeCaregiverSiteLookup struct{ byName map[string]uuid.UUID }

func (f fakeCaregiverSiteLookup) GetByName(ctx context.Context, name string) (*SiteRef, error) {
	id, ok := f.byName[name]
	if !ok {
		return nil, errCaregiverImportTestNotFound{}
	}
	return &SiteRef{ID: id, Name: name}, nil
}

type errCaregiverImportTestNotFound struct{}

func (errCaregiverImportTestNotFound) Error() string { return "not found" }

// xlsxReader 依表頭與逐列字串值組出一份真實 .xlsx 位元組，供測試以既有 ExcelAdapter 解析，
// 對齊本模組僅支援 .xlsx 匯入格式的限制。
func xlsxReader(t *testing.T, header []string, rows ...[]string) *bytes.Reader {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	sheetName := f.GetSheetName(0)

	for c, h := range header {
		cell, err := excelize.CoordinatesToCellName(c+1, 1)
		require.NoError(t, err)
		require.NoError(t, f.SetCellValue(sheetName, cell, h))
	}
	for r, row := range rows {
		for c, v := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+2)
			require.NoError(t, err)
			require.NoError(t, f.SetCellValue(sheetName, cell, v))
		}
	}

	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	return bytes.NewReader(buf.Bytes())
}

var caregiverHeader = []string{"單位", "姓名", "類型", "聯絡方式", "備註"}

func TestParseCaregivers_SkipsRowMissingName(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "", "個管", "0912-000-000", ""},
		[]string{"竹南日照單位", "王大明", "個管", "0987-000-000", "行動自如"},
	), "upload.xlsx")

	require.NoError(t, err)
	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 1, preview.ValidRows)
	assert.Equal(t, 1, preview.ErrorRows)
	require.Len(t, preview.Errors, 1)
	assert.Equal(t, 2, preview.Errors[0].RowIndex, "姓名缺漏的第 2 列應歸入 Errors 而不進入可匯入列")
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, "王大明", preview.Rows[0].Name)
}

func TestParseCaregivers_IgnoresFullyBlankRow(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"", "", "", "", ""},
		[]string{"竹南日照單位", "王大明", "個管", "0987-000-000", "行動自如"},
	), "upload.xlsx")

	require.NoError(t, err)
	assert.Equal(t, 1, preview.TotalRows, "全空白列不應計入總筆數")
	assert.Equal(t, 1, preview.ValidRows)
	assert.Equal(t, 0, preview.ErrorRows, "全空白列應直接忽略，不應歸入錯誤列")
	assert.Empty(t, preview.Errors)
}

func TestParseCaregivers_SkipsRowWithMissingOrInvalidType(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "王大明", "", "0987-000-000", "行動自如"},
		[]string{"竹南日照單位", "陳小華", "居服員", "0987-000-000", ""},
		[]string{"竹南日照單位", "李美玲", "個管", "0987-000-000", ""},
	), "upload.xlsx")

	require.NoError(t, err)
	assert.Equal(t, 3, preview.TotalRows)
	assert.Equal(t, 1, preview.ValidRows, "類型缺漏或不是個管／專護的列都應略過")
	require.Len(t, preview.Errors, 2)
	assert.Equal(t, "類型", preview.Errors[0].Field)
	assert.Equal(t, "類型", preview.Errors[1].Field)
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, "李美玲", preview.Rows[0].Name)
	assert.Equal(t, CaregiverTypeCaseManager, preview.Rows[0].Type)
}

func TestParseCaregivers_KeepsRawSiteNameWhenSiteNotFound(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{byName: map[string]uuid.UUID{}}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"查無此單位", "陳小華", "專護", "0912-345-678", "熟悉輪椅移位"},
	), "upload.xlsx")

	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	row := preview.Rows[0]
	assert.Nil(t, row.SiteID, "單位比對不到時應保留 SiteID 為 nil")
	assert.Equal(t, "查無此單位", row.SiteName)
	assert.Equal(t, CaregiverTypeSpecialist, row.Type)
	assert.NotEmpty(t, row.WarningMessage)
}

func TestParseCaregivers_FlagsMissingContactAndNotesAsWarningNotError(t *testing.T) {
	siteID := uuid.New()
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{byName: map[string]uuid.UUID{"竹南日照單位": siteID}}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "王大明", "個管", "", ""},
	), "upload.xlsx")

	require.NoError(t, err)
	assert.Equal(t, 1, preview.ValidRows, "聯絡方式與備註缺漏仍為合法可匯入列")
	assert.Equal(t, 0, preview.ErrorRows)
	require.Len(t, preview.Rows, 1)
	assert.Contains(t, preview.Rows[0].WarningMessage, "聯絡方式")
	assert.Contains(t, preview.Rows[0].WarningMessage, "備註")
}

func TestParseCaregivers_RejectsNonExcelUpload(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	_, err := svc.ParseCaregivers(context.Background(), bytes.NewReader([]byte("單位,姓名,類型,聯絡方式,備註\n竹南日照單位,王大明,個管,,")), "upload.csv")

	assert.Error(t, err, "僅支援 .xlsx 匯入，CSV 上傳應回傳錯誤")
}

func TestParseCaregivers_FlagsDuplicateByName(t *testing.T) {
	store := newFakeCaregiverStore()
	existingID := uuid.New()
	store.byID[existingID] = &Caregiver{ID: existingID, Name: "王大明", Type: CaregiverTypeCaseManager}
	svc := NewCaregiverService(store, fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "王大明", "個管", "0987-000-000", "行動自如"},
	), "upload.xlsx")

	require.NoError(t, err)
	require.Len(t, preview.Rows, 1, "重複人員不擋匯入，仍為合法可匯入列")
	row := preview.Rows[0]
	assert.True(t, row.IsDuplicate)
	assert.Equal(t, existingID, *row.DuplicateCaregiverID)
	assert.Contains(t, row.WarningMessage, "重複照護人員")
}

func TestCommitCaregivers_SkipsDuplicateRowUnlessIncluded(t *testing.T) {
	store := newFakeCaregiverStore()
	svc := NewCaregiverService(store, fakeCaregiverSiteLookup{}, nil, nil)
	dupID := uuid.New()
	preview := &CaregiverImportPreviewResult{
		Rows: []CaregiverImportRowResult{
			{RowIndex: 2, Name: "王大明", IsDuplicate: true, DuplicateCaregiverID: &dupID, DuplicateCaregiverName: "王大明"},
		},
	}

	skipped, err := svc.CommitCaregivers(context.Background(), preview, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, skipped.ImportedCount)
	require.Len(t, skipped.SkippedRows, 1, "未勾選的重複列應略過")
	assert.Equal(t, 2, skipped.SkippedRows[0].RowIndex)

	included, err := svc.CommitCaregivers(context.Background(), preview, map[string]bool{"legacy:2": true})
	require.NoError(t, err)
	assert.Equal(t, 1, included.ImportedCount, "已勾選的重複列應正常匯入")
	assert.Empty(t, included.SkippedRows)
}

func TestCommitCaregivers_ImportsRowsAndReportsWarningsByField(t *testing.T) {
	store := newFakeCaregiverStore()
	svc := NewCaregiverService(store, fakeCaregiverSiteLookup{}, nil, nil)

	preview := &CaregiverImportPreviewResult{
		Rows: []CaregiverImportRowResult{
			{RowIndex: 2, Name: "查無單位者", SiteName: "查無此單位", WarningMessage: "單位未比對到"},
			{RowIndex: 3, Name: "缺聯絡方式者"},
		},
		Errors: []CaregiverImportErrorItem{
			{RowIndex: 4, Message: "姓名：未填寫，本列已略過"},
		},
	}

	result, err := svc.CommitCaregivers(context.Background(), preview, nil)

	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount)
	require.Len(t, result.SkippedRows, 1, "姓名缺漏列應歸入略過清單並回報原因")
	assert.Equal(t, 4, result.SkippedRows[0].RowIndex)

	var siteWarning, contactWarning, notesWarning bool
	for _, w := range result.Warnings {
		switch w.Field {
		case "site":
			siteWarning = true
			assert.Equal(t, 2, w.RowIndex)
		case "contact":
			contactWarning = true
		case "notes":
			notesWarning = true
		}
	}
	assert.True(t, siteWarning, "單位未比對到的列應標記 field=site")
	assert.True(t, contactWarning, "聯絡方式缺漏的列應標記 field=contact")
	assert.True(t, notesWarning, "備註缺漏的列應標記 field=notes")

	assert.Len(t, store.byID, 2, "兩列都應建立資料，僅姓名缺漏的列才略過")
}
