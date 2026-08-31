package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCaseID = "11111111-1111-1111-1111-111111111111"
	testFormID = "22222222-2222-2222-2222-222222222222"
)

// stubStore 只回應解析階段會用到的讀取；寫入方法在解析路徑上不應被呼叫。
type stubStore struct {
	existing []ColumnMapping
	form     *ReportForm
}

func (s *stubStore) ListForms(context.Context) ([]ReportForm, error) { return nil, nil }
func (s *stubStore) GetForm(context.Context, uuid.UUID) (*ReportForm, error) {
	return s.form, nil
}
func (s *stubStore) CreateForm(_ context.Context, id, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return id, nil
}
func (s *stubStore) DeleteForm(context.Context, uuid.UUID) error { return nil }
func (s *stubStore) ListColumnsWithMapping(_ context.Context, _, mappingStatus string) ([]ColumnMapping, error) {
	if mappingStatus == "" {
		return s.existing, nil
	}
	var out []ColumnMapping
	for _, c := range s.existing {
		if c.MappingStatus == mappingStatus {
			out = append(out, c)
		}
	}
	return out, nil
}
func (s *stubStore) UpsertColumns(context.Context, uuid.UUID, []ColumnDraft) error { return nil }
func (s *stubStore) UpdateColumnMappingByID(context.Context, string, string, *string, *int16) error {
	return nil
}
func (s *stubStore) UpdateColumnMappingByHeader(context.Context, uuid.UUID, string, string, *string, *int16) error {
	return nil
}
func (s *stubStore) MarkImported(context.Context, uuid.UUID, time.Time) error { return nil }

type stubExcel struct{ table [][]string }

func (s stubExcel) ReadTables([]byte) ([][][]string, []string, error) {
	return [][][]string{s.table}, []string{"司機接送匯報"}, nil
}

type stubCases struct{ list []CaseRef }

func (s stubCases) ListActiveCases(context.Context) ([]CaseRef, error) { return s.list, nil }

type stubDrivers struct{ known map[string]DriverRef }

func (s stubDrivers) GetByNameNormalized(_ context.Context, nameNorm string) (*DriverRef, error) {
	if d, ok := s.known[nameNorm]; ok {
		return &d, nil
	}
	return nil, nil
}

func newTestService(table [][]string, existing []ColumnMapping) *DriverReportService {
	vehicleID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	return NewDriverReportService(
		&stubStore{
			existing: existing,
			form: &ReportForm{
				ID:                 uuid.MustParse(testFormID),
				VehicleID:          vehicleID,
				VehicleDisplayName: "竹南2車",
				Title:              "竹南2車接送匯報",
			},
		},
		stubExcel{table: table},
		nil,
		stubCases{list: []CaseRef{{ID: testCaseID, Name: "吳桂", NameNormalized: "吳桂"}}},
		stubDrivers{known: map[string]DriverRef{"林彥衡": {ID: uuid.New(), Name: "林彥衡"}}},
		nil,
		nil,
		nil,
	)
}

func sampleTable() [][]string {
	return [][]string{
		{"民國日期", "駕駛人", "1.吳桂(去程竹3) [去程]", "1.吳桂(去程竹3) [回程]", "備註"},
		{"1150302", "林彥衡", "有坐", "沒坐", "無"},
		{"115/3/3", "林彥衡", "有坐", "有坐", ""},
		{"", "", "", "", ""},
		{"壞掉的日期", "杜瑞明", "有坐", "有坐", "測試"},
	}
}

func TestParseReportDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "民國七碼", input: "1150302", want: "2026-03-02"},
		{name: "民國七碼含空白", input: " 1150302 ", want: "2026-03-02"},
		{name: "民國斜線", input: "115/3/2", want: "2026-03-02"},
		{name: "民國減號補零", input: "115-03-02", want: "2026-03-02"},
		{name: "西元後備寫法", input: "2026-03-02", want: "2026-03-02"},
		{name: "空字串", input: "", wantErr: true},
		{name: "非日期文字", input: "壞掉的日期", wantErr: true},
		{name: "不存在的日期", input: "1150230", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseReportDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Format("2006-01-02"))
		})
	}
}

func TestParseDriverReport_ColumnsAndRows(t *testing.T) {
	svc := newTestService(sampleTable(), nil)

	result, err := svc.ParseDriverReport(context.Background(), uuid.MustParse(testFormID), strings.NewReader("x"), "")
	require.NoError(t, err)

	// 兩個個案趟次欄；第一欄日期、第二欄駕駛人、最後一欄備註都不算欄位對應對象
	require.Len(t, result.Columns, 2)
	assert.Equal(t, 3, result.Columns[0].ColumnIndex)
	assert.Equal(t, "outbound", result.Columns[0].Direction)
	assert.Equal(t, "inbound", result.Columns[1].Direction)
	assert.Equal(t, 2, result.UnmappedColumns)

	require.NotNil(t, result.Columns[0].SuggestedCaseID)
	assert.Equal(t, testCaseID, *result.Columns[0].SuggestedCaseID)
	require.NotNil(t, result.Columns[0].SuggestedLegSeq)
	assert.Equal(t, int16(1), *result.Columns[0].SuggestedLegSeq)
	require.NotNil(t, result.Columns[1].SuggestedLegSeq)
	assert.Equal(t, int16(2), *result.Columns[1].SuggestedLegSeq)

	// 空白日期列被略過不計；壞掉的日期列計為錯誤列
	assert.Equal(t, 3, result.TotalRows)
	assert.Equal(t, 2, result.ValidRows)
	assert.Equal(t, 1, result.ErrorRows)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "民國日期", result.Errors[0].Field)

	require.Len(t, result.PreviewRows, 3)
	assert.Equal(t, "2026-03-02", result.PreviewRows[0].ServiceDate)
	assert.Equal(t, "林彥衡", result.PreviewRows[0].DriverName)
	assert.Equal(t, "無", result.PreviewRows[0].Remark)
	assert.Equal(t, 1, result.PreviewRows[0].BoardedCount)
	assert.Equal(t, 1, result.PreviewRows[0].AbsentCount)

	// 逐欄統計：去程兩天都有坐，回程一天沒坐一天有坐
	assert.Equal(t, 2, result.Columns[0].BoardedCount)
	assert.Equal(t, 0, result.Columns[0].AbsentCount)
	assert.Equal(t, 1, result.Columns[1].BoardedCount)
	assert.Equal(t, 1, result.Columns[1].AbsentCount)
}

func TestParseDriverReport_RowsOutsideDeclaredMonthAreErrorRowsNotWholeFileRejection(t *testing.T) {
	// 月份不符不再整份拒絕：解析仍要成功回傳，讓使用者在預覽畫面看得到比對結果，
	// 自行決定要調整宣告月份還是忽略那幾列後繼續匯入。
	svc := newTestService(sampleTable(), nil)

	result, err := svc.ParseDriverReport(context.Background(), uuid.MustParse(testFormID), strings.NewReader("x"), "2026-04")
	require.NoError(t, err)

	assert.Equal(t, 0, result.ValidRows, "sampleTable 的日期都在三月，宣告 2026-04 時沒有列落在月份內")
	assert.Equal(t, 3, result.ErrorRows)
	require.Len(t, result.PreviewRows, 3)
	assert.Contains(t, result.PreviewRows[0].ErrorMessage, "不屬於宣告匯入的 2026-04")
	assert.Equal(t, "2026-03-02", result.PreviewRows[0].ServiceDate, "月份不符的列仍保留已解析出的服務日期供畫面顯示")
}

func TestParseDriverReport_UnknownDriverIsWarningNotError(t *testing.T) {
	table := sampleTable()
	table[1][1] = "查無此人"
	svc := newTestService(table, nil)

	result, err := svc.ParseDriverReport(context.Background(), uuid.MustParse(testFormID), strings.NewReader("x"), "")
	require.NoError(t, err)

	assert.Equal(t, 2, result.ValidRows)
	require.NotEmpty(t, result.Warnings)
	assert.Contains(t, result.Warnings[0].Message, "未比對到司機主檔")
	assert.Empty(t, result.PreviewRows[0].DriverID)
}

func TestParseDriverReport_KeepsExistingMapping(t *testing.T) {
	caseID := testCaseID
	caseName := "吳桂"
	legSeq := int16(1)
	svc := newTestService(sampleTable(), []ColumnMapping{{
		ID:            "col-1",
		ColumnHeader:  "1.吳桂(去程竹3) [去程]",
		MappingStatus: "mapped",
		CaseID:        &caseID,
		CaseName:      &caseName,
		LegSeq:        &legSeq,
	}})

	result, err := svc.ParseDriverReport(context.Background(), uuid.MustParse(testFormID), strings.NewReader("x"), "")
	require.NoError(t, err)

	assert.Equal(t, "mapped", result.Columns[0].MappingStatus)
	assert.Equal(t, "col-1", result.Columns[0].ColumnID)
	assert.Equal(t, 1, result.UnmappedColumns, "只剩回程欄待對應")
}

func TestParseDriverReport_AcceptsLeadingTimestampColumn(t *testing.T) {
	// Google 表單原始匯出檔會在日期欄前多一欄「時間戳記」；日期／駕駛人欄應改用
	// 內容比對定位，而不是假設固定在第 0、1 欄。
	table := [][]string{
		{"時間戳記", "今天日期", "今日駕駛人", "1.吳桂(去程竹3) [去程]", "問題回報"},
		{"46084", "1150302", "林彥衡", "有坐", "備註內容"},
	}
	svc := newTestService(table, nil)

	result, err := svc.ParseDriverReport(context.Background(), uuid.MustParse(testFormID), strings.NewReader("x"), "")
	require.NoError(t, err)

	require.Len(t, result.PreviewRows, 1)
	assert.Equal(t, "2026-03-02", result.PreviewRows[0].ServiceDate)
	assert.Equal(t, "林彥衡", result.PreviewRows[0].DriverName)
	assert.Equal(t, "備註內容", result.PreviewRows[0].Remark)
	assert.Equal(t, 1, result.PreviewRows[0].BoardedCount)
}

func TestParseDriverReport_IgnoresColumnsAfterRemark(t *testing.T) {
	// 備註欄之後若還殘留欄位（如表單後續編修累加的舊題目），一律忽略，不當成個案欄匯入。
	table := [][]string{
		{"民國日期", "駕駛人", "1.吳桂(去程竹3) [去程]", "備註", "10.李吳素娥 [去程]"},
		{"1150302", "林彥衡", "有坐", "無", "有坐"},
	}
	svc := newTestService(table, nil)

	result, err := svc.ParseDriverReport(context.Background(), uuid.MustParse(testFormID), strings.NewReader("x"), "")
	require.NoError(t, err)

	require.Len(t, result.Columns, 1)
	assert.Equal(t, "1.吳桂(去程竹3) [去程]", result.Columns[0].ColumnHeader)
	assert.Equal(t, "無", result.PreviewRows[0].Remark)
}

func TestParseDriverReport_RejectsWrongHeader(t *testing.T) {
	table := [][]string{
		{"備註", "其他欄位"},
		{"", ""},
	}
	svc := newTestService(table, nil)

	_, err := svc.ParseDriverReport(context.Background(), uuid.MustParse(testFormID), strings.NewReader("x"), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "找不到匯報表表頭")
}

func TestParseYearMonth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		declared bool
		wantErr  bool
	}{
		{name: "正常月份", input: "2026-03", want: "2026-03-01", declared: true},
		{name: "前後空白", input: " 2026-03 ", want: "2026-03-01", declared: true},
		{name: "空字串代表未宣告", input: "", declared: false},
		{name: "月份超出範圍", input: "2026-13", wantErr: true},
		{name: "月份為零", input: "2026-00", wantErr: true},
		{name: "月份未補零", input: "2026-3", wantErr: true},
		{name: "年份位數不足", input: "26-01", wantErr: true},
		{name: "斜線分隔", input: "2026/03", wantErr: true},
		{name: "帶到日", input: "2026-03-01", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, declared, err := parseYearMonth(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidYearMonth)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.declared, declared)
			if tt.declared {
				assert.Equal(t, tt.want, got.Format("2006-01-02"))
			}
		})
	}
}

func TestDaysInMonth(t *testing.T) {
	tests := []struct {
		name      string
		yearMonth string
		wantCount int
		wantLast  string
	}{
		{name: "閏年二月", yearMonth: "2024-02", wantCount: 29, wantLast: "2024-02-29"},
		{name: "平年二月", yearMonth: "2026-02", wantCount: 28, wantLast: "2026-02-28"},
		{name: "三十天的月份", yearMonth: "2026-04", wantCount: 30, wantLast: "2026-04-30"},
		{name: "十二月不跨年溢位", yearMonth: "2026-12", wantCount: 31, wantLast: "2026-12-31"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, declared, err := parseYearMonth(tt.yearMonth)
			require.NoError(t, err)
			require.True(t, declared)

			days := daysInMonth(start)

			require.Len(t, days, tt.wantCount)
			assert.Equal(t, tt.yearMonth+"-01", days[0].Format("2006-01-02"))
			assert.Equal(t, tt.wantLast, days[len(days)-1].Format("2006-01-02"))
		})
	}
}
