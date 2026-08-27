package service_test

import (
	"context"
	"testing"

	"ltc-system/apps/api/internal/adapter/google"
	"ltc-system/apps/api/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGoogleAdapter struct{}

func (m *mockGoogleAdapter) ListDriveSheets(ctx context.Context) ([]google.DriveFileItem, error) {
	return []google.DriveFileItem{
		{ID: "sheet_001", Name: "竹北一車每日接送回報 (回覆)"},
		{ID: "sheet_002", Name: "竹北二車每日接送回報 (回覆)"},
	}, nil
}

func (m *mockGoogleAdapter) GetSpreadsheetInfo(ctx context.Context, spreadsheetID string) (*google.SpreadsheetInfo, error) {
	return &google.SpreadsheetInfo{
		SpreadsheetID: spreadsheetID,
		Title:         "竹北一車每日接送回報 (回覆)",
		SheetTabs:     []string{"8月回報", "7月回報", "表單回覆 1"},
	}, nil
}

func (m *mockGoogleAdapter) ReadSheetRows(ctx context.Context, spreadsheetID string, tabName string) ([][]interface{}, error) {
	return [][]interface{}{
		{"時間戳記", "今天日期", "今日駕駛人", "蔡曾切（去）"},
		{"2026/08/25 15:30:00", "2026-08-25", "陳司機", "有坐"},
	}, nil
}

func TestFormService_GoogleOperations(t *testing.T) {
	ctx := context.Background()
	svc := service.NewFormService(nil, &mockGoogleAdapter{})

	t.Run("列出 Google Drive 試算表", func(t *testing.T) {
		files, err := svc.ListGoogleDriveFiles(ctx)
		require.NoError(t, err)
		assert.Len(t, files, 2)
		assert.Equal(t, "竹北一車每日接送回報 (回覆)", files[0].Name)
	})

	t.Run("解析 Google 試算表結構與分頁", func(t *testing.T) {
		info, err := svc.InspectGoogleSheet(ctx, "https://docs.google.com/spreadsheets/d/sheet_001/edit")
		require.NoError(t, err)
		assert.Equal(t, "sheet_001", info.SpreadsheetID)
		assert.Equal(t, "竹北一車每日接送回報 (回覆)", info.Title)
		assert.Equal(t, []string{"8月回報", "7月回報", "表單回覆 1"}, info.SheetTabs)
		assert.Contains(t, info.PreviewHeaders, "蔡曾切（去）")
	})

	t.Run("建立表單關聯", func(t *testing.T) {
		req := service.CreateFormAssociationRequest{
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
		res, err := svc.SyncForm(ctx, "sheet_001", &service.SyncFormOptions{
			Month:    "2026-08",
			SheetTab: "8月回報",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, res["syncedRows"])
		assert.Equal(t, "2026-08", res["month"])
	})
}
