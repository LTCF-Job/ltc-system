package infra

import (
	"context"

	"ltc-system/apps/api/internal/modules/formsync/app"
	"ltc-system/apps/api/internal/modules/formsync/infra/google"
)

// GoogleClient 把 Google SDK 的回傳型別轉成 formsync 自己的型別，讓 app 層不需
// 認識任何 vendor SDK 結構。
type GoogleClient struct {
	adapter google.Adapter
}

// NewGoogleClient 建立 GoogleClient；adapter 為 nil 時回傳 nil，維持離線啟動行為。
func NewGoogleClient(adapter google.Adapter) *GoogleClient {
	if adapter == nil {
		return nil
	}
	return &GoogleClient{adapter: adapter}
}

// ListDriveSheets 列出雲端硬碟中的試算表。
func (c *GoogleClient) ListDriveSheets(ctx context.Context) ([]app.DriveFile, error) {
	items, err := c.adapter.ListDriveSheets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]app.DriveFile, 0, len(items))
	for _, it := range items {
		out = append(out, app.DriveFile{ID: it.ID, Name: it.Name, MimeType: it.MimeType, ModifiedTime: it.ModifiedTime})
	}
	return out, nil
}

// GetSpreadsheetInfo 取得試算表結構與分頁資訊。
func (c *GoogleClient) GetSpreadsheetInfo(ctx context.Context, spreadsheetID, accessToken string) (*app.SpreadsheetInfo, error) {
	info, err := c.adapter.GetSpreadsheetInfo(ctx, spreadsheetID, accessToken)
	if err != nil {
		return nil, err
	}
	return &app.SpreadsheetInfo{SpreadsheetID: info.SpreadsheetID, Title: info.Title, SheetTabs: info.SheetTabs}, nil
}

// ReadSheetRows 讀取指定分頁的所有列。
func (c *GoogleClient) ReadSheetRows(ctx context.Context, spreadsheetID, tabName, accessToken string) ([][]interface{}, error) {
	return c.adapter.ReadSheetRows(ctx, spreadsheetID, tabName, accessToken)
}
