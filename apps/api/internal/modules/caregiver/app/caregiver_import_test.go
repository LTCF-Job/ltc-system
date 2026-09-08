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
// err 模擬查詢本身故障，與「查無單位」是不同的路徑。
type fakeCaregiverSiteLookup struct {
	byName map[string]uuid.UUID
	err    error
}

func (f fakeCaregiverSiteLookup) GetByName(ctx context.Context, name string) (*SiteRef, error) {
	if f.err != nil {
		return nil, f.err
	}
	id, ok := f.byName[name]
	if !ok {
		return nil, ErrCaregiverSiteNotFound
	}
	return &SiteRef{ID: id, Name: name}, nil
}

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

func TestParseCaregivers_KeepsRowMissingNameAsPending(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "", "個管", "0912-000-000", ""},
		[]string{"竹南日照單位", "王大明", "個管", "0987-000-000", "行動自如"},
	), "upload.xlsx")

	require.NoError(t, err)
	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 2, preview.ValidRows, "姓名缺漏不再擋列，改以空白建立並列入待維護")
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Empty(t, preview.Errors)
	require.Len(t, preview.Rows, 2)
	assert.Equal(t, "", preview.Rows[0].Name)
	assert.Contains(t, preview.Rows[0].WarningMessage, "姓名")
	assert.Equal(t, "王大明", preview.Rows[1].Name)
	assert.Empty(t, preview.Rows[1].WarningMessage)
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

func TestParseCaregivers_KeepsRowWithMissingOrInvalidTypeAsPending(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "王大明", "", "0987-000-000", "行動自如"},
		[]string{"竹南日照單位", "陳小華", "居服員", "0987-000-000", ""},
		[]string{"竹南日照單位", "李美玲", "個管", "0987-000-000", ""},
		[]string{"竹南日照單位", "張大千", "照專", "0987-000-000", ""},
		[]string{"竹南日照單位", "何專護", "專護", "0987-000-000", ""},
	), "upload.xlsx")

	require.NoError(t, err)
	assert.Equal(t, 5, preview.TotalRows)
	assert.Equal(t, 5, preview.ValidRows, "類型缺漏或不是個管／照專都改以空白建立並列入待維護")
	assert.Empty(t, preview.Errors)
	require.Len(t, preview.Rows, 5)
	assert.Equal(t, "", preview.Rows[0].Type)
	assert.Contains(t, preview.Rows[0].WarningMessage, "類型")
	assert.Equal(t, "", preview.Rows[1].Type, "「居服員」不是固定選項，同樣存成空白")
	assert.Contains(t, preview.Rows[1].WarningMessage, "類型")
	assert.Equal(t, "李美玲", preview.Rows[2].Name)
	assert.Equal(t, CaregiverTypeCaseManager, preview.Rows[2].Type)
	assert.Equal(t, "張大千", preview.Rows[3].Name)
	assert.Equal(t, CaregiverTypeSpecialist, preview.Rows[3].Type)
	assert.Equal(t, "何專護", preview.Rows[4].Name)
	assert.Equal(t, CaregiverTypeSpecialist, preview.Rows[4].Type, "向後相容舊稱專護")
}

func TestParseCaregivers_LeavesSiteUnlinkedWhenSiteNotFound(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{byName: map[string]uuid.UUID{}}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"查無此單位", "陳小華", "專護", "0912-345-678", "熟悉輪椅移位"},
	), "upload.xlsx")

	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	row := preview.Rows[0]
	assert.Nil(t, row.SiteID, "單位比對不到時應保留 SiteID 為 nil")
	assert.Equal(t, CaregiverTypeSpecialist, row.Type)
	assert.Empty(t, row.WarningMessage, "單位比對不到只留白，不再列為待維護或產生警告")
}

func TestParseCaregivers_DoesNotWarnOnMissingContactOrNotes(t *testing.T) {
	siteID := uuid.New()
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{byName: map[string]uuid.UUID{"竹南日照單位": siteID}}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "王大明", "個管", "", ""},
	), "upload.xlsx")

	require.NoError(t, err)
	assert.Equal(t, 1, preview.ValidRows)
	assert.Equal(t, 0, preview.ErrorRows)
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, &siteID, preview.Rows[0].SiteID)
	assert.Empty(t, preview.Rows[0].WarningMessage, "聯絡方式與備註缺漏不再算待維護，不產生警告")
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

// 姓名為空時若仍查重，ILIKE '%%' 會撈回任意資料列，且正規化後的空字串會與既有空姓名
// 資料互相命中，導致每一列都被判為重複而預設不匯入。
func TestParseCaregivers_SkipsDuplicateLookupWhenNameEmpty(t *testing.T) {
	store := newFakeCaregiverStore()
	existingID := uuid.New()
	store.byID[existingID] = &Caregiver{ID: existingID, Name: "", Type: CaregiverTypeCaseManager, Status: "active"}
	svc := NewCaregiverService(store, fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	preview, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "", "個管", "0987-000-000", ""},
	), "upload.xlsx")

	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.False(t, preview.Rows[0].IsDuplicate, "姓名為空的列不應進行重複比對")
}

// 單位查詢失敗與「查無單位」是兩件事：前者必須中止預覽，不能被降級成靜默略過。
func TestParseCaregivers_AbortsWhenSiteLookupFailsWithRealError(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{err: assert.AnError}, testExcelReader{}, nil)

	_, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"竹南日照單位", "王大明", "個管", "0987-000-000", "行動自如"},
	), "upload.xlsx")

	assert.ErrorIs(t, err, assert.AnError)
}

func TestParseCaregivers_AbortsWhenDuplicateLookupFails(t *testing.T) {
	store := newFakeCaregiverStore()
	store.listErr = assert.AnError
	svc := NewCaregiverService(store, fakeCaregiverSiteLookup{}, testExcelReader{}, nil)

	_, err := svc.ParseCaregivers(context.Background(), xlsxReader(t, caregiverHeader,
		[]string{"查無此單位", "王大明", "個管", "0987-000-000", "行動自如"},
	), "upload.xlsx")

	assert.ErrorIs(t, err, assert.AnError)
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
			// 單位比對不到：SiteName 有值但不得寫入 SiteNameRaw，也不該產生警告
			{RowIndex: 2, Name: "查無單位者", Type: CaregiverTypeCaseManager, SiteName: "查無此單位"},
			{RowIndex: 3, Name: "", Type: CaregiverTypeSpecialist},
			{RowIndex: 4, Name: "缺類型者", Type: ""},
		},
	}

	result, err := svc.CommitCaregivers(context.Background(), preview, nil)

	require.NoError(t, err)
	assert.Equal(t, 3, result.ImportedCount, "姓名或類型缺漏的列同樣要建立資料")
	assert.Empty(t, result.SkippedRows)

	var nameWarning, typeWarning bool
	for _, w := range result.Warnings {
		switch w.Field {
		case "name":
			nameWarning = true
			assert.Equal(t, 3, w.RowIndex)
		case "type":
			typeWarning = true
			assert.Equal(t, 4, w.RowIndex)
		default:
			t.Fatalf("不應再產生 %q 欄位的警告", w.Field)
		}
	}
	assert.True(t, nameWarning, "姓名缺漏的列應標記 field=name")
	assert.True(t, typeWarning, "類型缺漏的列應標記 field=type")

	require.Len(t, store.byID, 3)
	for _, c := range store.byID {
		assert.Empty(t, c.SiteNameRaw, "單位比對不到時只留白，不得保留原始名稱")
	}
}
