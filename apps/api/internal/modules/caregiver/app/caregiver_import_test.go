package app

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func csvReader(header string, rows ...string) *strings.Reader {
	return strings.NewReader(header + "\n" + strings.Join(rows, "\n"))
}

func TestParseCaregivers_SkipsRowMissingName(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{}, nil, nil)

	preview, err := svc.ParseCaregivers(context.Background(), csvReader(
		"單位,姓名,類型,聯絡方式,備註",
		"竹南日照據點,,個管,0912-000-000,",
		"竹南日照據點,王大明,個管,0987-000-000,行動自如",
	), "upload.csv")

	require.NoError(t, err)
	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 1, preview.ValidRows)
	assert.Equal(t, 1, preview.ErrorRows)
	require.Len(t, preview.Errors, 1)
	assert.Equal(t, 2, preview.Errors[0].RowIndex, "姓名缺漏的第 2 列應歸入 Errors 而不進入可匯入列")
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, "王大明", preview.Rows[0].Name)
}

func TestParseCaregivers_SkipsRowWithMissingOrInvalidType(t *testing.T) {
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{}, nil, nil)

	preview, err := svc.ParseCaregivers(context.Background(), csvReader(
		"單位,姓名,類型,聯絡方式,備註",
		"竹南日照據點,王大明,,0987-000-000,行動自如",
		"竹南日照據點,陳小華,居服員,0987-000-000,",
		"竹南日照據點,李美玲,個管,0987-000-000,",
	), "upload.csv")

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
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{byName: map[string]uuid.UUID{}}, nil, nil)

	preview, err := svc.ParseCaregivers(context.Background(), csvReader(
		"單位,姓名,類型,聯絡方式,備註",
		"查無此據點,陳小華,專護,0912-345-678,熟悉輪椅移位",
	), "upload.csv")

	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	row := preview.Rows[0]
	assert.Nil(t, row.SiteID, "單位比對不到時應保留 SiteID 為 nil")
	assert.Equal(t, "查無此據點", row.SiteName)
	assert.Equal(t, CaregiverTypeSpecialist, row.Type)
	assert.NotEmpty(t, row.WarningMessage)
}

func TestParseCaregivers_FlagsMissingContactAndNotesAsWarningNotError(t *testing.T) {
	siteID := uuid.New()
	svc := NewCaregiverService(newFakeCaregiverStore(), fakeCaregiverSiteLookup{byName: map[string]uuid.UUID{"竹南日照據點": siteID}}, nil, nil)

	preview, err := svc.ParseCaregivers(context.Background(), csvReader(
		"單位,姓名,類型,聯絡方式,備註",
		"竹南日照據點,王大明,個管,,",
	), "upload.csv")

	require.NoError(t, err)
	assert.Equal(t, 1, preview.ValidRows, "聯絡方式與備註缺漏仍為合法可匯入列")
	assert.Equal(t, 0, preview.ErrorRows)
	require.Len(t, preview.Rows, 1)
	assert.Contains(t, preview.Rows[0].WarningMessage, "聯絡方式")
	assert.Contains(t, preview.Rows[0].WarningMessage, "備註")
}

func TestCommitCaregivers_ImportsRowsAndReportsWarningsByField(t *testing.T) {
	store := newFakeCaregiverStore()
	svc := NewCaregiverService(store, fakeCaregiverSiteLookup{}, nil, nil)

	preview := &CaregiverImportPreviewResult{
		Rows: []CaregiverImportRowResult{
			{RowIndex: 2, Name: "查無單位者", SiteName: "查無此據點", WarningMessage: "單位未比對到"},
			{RowIndex: 3, Name: "缺聯絡方式者"},
		},
		Errors: []CaregiverImportErrorItem{
			{RowIndex: 4, Message: "姓名：未填寫，本列已略過"},
		},
	}

	result, err := svc.CommitCaregivers(context.Background(), preview)

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
