package app_test

import (
	"context"
	"testing"

	"ltc-system/apps/api/internal/modules/formsync/app"
	"ltc-system/apps/api/internal/modules/formsync/infra"
	"ltc-system/apps/api/internal/modules/formsync/infra/google"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGoogleAdapter struct {
	lastInfoToken string
	lastRowsToken string
}

func (m *mockGoogleAdapter) ListDriveSheets(ctx context.Context) ([]app.DriveFile, error) {
	return []app.DriveFile{
		{ID: "sheet_001", Name: "竹北一車每日接送回報 (回覆)"},
		{ID: "sheet_002", Name: "竹北二車每日接送回報 (回覆)"},
	}, nil
}

func (m *mockGoogleAdapter) GetSpreadsheetInfo(ctx context.Context, spreadsheetID string, accessToken string) (*app.SpreadsheetInfo, error) {
	m.lastInfoToken = accessToken
	return &app.SpreadsheetInfo{
		SpreadsheetID: spreadsheetID,
		Title:         "竹北一車每日接送回報 (回覆)",
		SheetTabs:     []string{"8月回報", "7月回報", "表單回覆 1"},
	}, nil
}

func (m *mockGoogleAdapter) ReadSheetRows(ctx context.Context, spreadsheetID string, tabName string, accessToken string) ([][]interface{}, error) {
	m.lastRowsToken = accessToken
	return [][]interface{}{
		{"時間戳記", "今天日期", "今日駕駛人", "蔡曾切（去）"},
		{"2026/08/25 15:30:00", "2026-08-25", "陳司機", "有坐"},
	}, nil
}

// TestFormService_GoogleClientNilChain 重現 cmd/server/main.go 未設定 GOOGLE_SA_JSON 時的真實建構鏈：
// nil 的具體型別指標若直接指派給介面參數，會變成「非 nil 介面、nil 內容」，讓 s.googleCli == nil
// 的判斷永遠不成立，導致離線降級分支變成死碼、實際仍會嘗試呼叫底層方法。此測試確保鏈路中每一層
// 都正確傳遞「真正的 nil 介面」。
func TestFormService_GoogleClientNilChain(t *testing.T) {
	ctx := context.Background()

	googleCli, err := google.NewClient(ctx, "")
	require.NoError(t, err)
	require.Nil(t, googleCli)

	var googleAdapter google.Adapter
	if googleCli != nil {
		googleAdapter = googleCli
	}

	var formGoogleClient app.GoogleClient
	if gc := infra.NewGoogleClient(googleAdapter); gc != nil {
		formGoogleClient = gc
	}

	svc := app.NewFormService(nil, formGoogleClient)

	_, err = svc.ListGoogleDriveFiles(ctx)
	assert.ErrorIs(t, err, app.ErrGoogleClientUnavailable)

	_, err = svc.InspectGoogleSheet(ctx, "https://docs.google.com/spreadsheets/d/demo-id/edit", "")
	assert.ErrorIs(t, err, app.ErrGoogleClientUnavailable)
}

func TestFormService_GoogleOperations(t *testing.T) {
	ctx := context.Background()
	adapter := &mockGoogleAdapter{}
	svc := app.NewFormService(nil, adapter)

	t.Run("列出 Google Drive 試算表", func(t *testing.T) {
		files, err := svc.ListGoogleDriveFiles(ctx)
		require.NoError(t, err)
		assert.Len(t, files, 2)
		assert.Equal(t, "竹北一車每日接送回報 (回覆)", files[0].Name)
	})

	t.Run("解析 Google 試算表結構與分頁", func(t *testing.T) {
		info, err := svc.InspectGoogleSheet(ctx, "https://docs.google.com/spreadsheets/d/sheet_001/edit", "user-token")
		require.NoError(t, err)
		assert.Equal(t, "sheet_001", info.SpreadsheetID)
		assert.Equal(t, "竹北一車每日接送回報 (回覆)", info.Title)
		assert.Equal(t, []string{"8月回報", "7月回報", "表單回覆 1"}, info.SheetTabs)
		assert.Contains(t, info.PreviewHeaders, "蔡曾切（去）")
		assert.Equal(t, "user-token", adapter.lastInfoToken)
		assert.Equal(t, "user-token", adapter.lastRowsToken)
	})

	t.Run("建立表單關聯", func(t *testing.T) {
		req := app.CreateFormAssociationRequest{
			Title:       "新竹北一車",
			SheetURL:    "https://docs.google.com/spreadsheets/d/sheet_001/edit",
			VehicleName: "竹北一車",
			Region:      "hsinchu",
			SheetTabs:   []string{"8月回報", "7月回報"},
			ActiveTab:   "8月回報",
		}
		item, err := svc.CreateFormAssociation(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, item.ID)
		assert.Equal(t, "新竹北一車", item.Title)
		assert.Equal(t, "8月回報", item.ActiveTab)
	})

	t.Run("表單同步操作", func(t *testing.T) {
		res, err := svc.SyncForm(ctx, "sheet_001", &app.SyncFormOptions{
			Month:         "2026-08",
			SheetTab:      "8月回報",
			AccessToken:   "user-token",
			SpreadsheetID: "sheet_001",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, res["syncedRows"])
		assert.Equal(t, "2026-08", res["month"])
		assert.Equal(t, "user-token", adapter.lastRowsToken)
	})
}
