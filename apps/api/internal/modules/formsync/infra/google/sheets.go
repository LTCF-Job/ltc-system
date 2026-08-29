package google

import (
	"context"
	"errors"
	"fmt"
	"ltc-system/apps/api/internal/modules/formsync/app"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// DriveFileItem 代表雲端硬碟中的試算表檔案。
type DriveFileItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType,omitempty"`
	ModifiedTime string `json:"modifiedTime,omitempty"`
}

// SpreadsheetInfo 代表試算表結構與分頁資訊。
type SpreadsheetInfo struct {
	SpreadsheetID string   `json:"spreadsheetId"`
	Title         string   `json:"title"`
	SheetTabs     []string `json:"sheetTabs"`
}

// Adapter 定義 Google 試算表與雲端硬碟的操作介面。
type Adapter interface {
	ListDriveSheets(ctx context.Context) ([]DriveFileItem, error)
	GetSpreadsheetInfo(ctx context.Context, spreadsheetID string, accessToken string) (*SpreadsheetInfo, error)
	ReadSheetRows(ctx context.Context, spreadsheetID string, tabName string, accessToken string) ([][]interface{}, error)
}

// Client 封裝 Google Drive 與 Sheets 服務。
type Client struct {
	driveSvc  *drive.Service
	sheetsSvc *sheets.Service
}

// NewClient 建立 Google API Client；saJSON 為空時代表未設定 Service Account 憑證，
// 回傳 nil 讓呼叫端視為「Google 功能不可用」，不得在此假裝建立成功後用假資料頂替。
func NewClient(ctx context.Context, saJSON string) (*Client, error) {
	if strings.TrimSpace(saJSON) == "" {
		return nil, nil
	}

	opt := option.WithCredentialsJSON([]byte(saJSON))

	driveSvc, err := drive.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize google drive service: %w", err)
	}

	sheetsSvc, err := sheets.NewService(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize google sheets service: %w", err)
	}

	return &Client{
		driveSvc:  driveSvc,
		sheetsSvc: sheetsSvc,
	}, nil
}

// ListDriveSheets 查詢 Service Account 可存取之試算表清單。
func (c *Client) ListDriveSheets(ctx context.Context) ([]DriveFileItem, error) {
	query := "mimeType = 'application/vnd.google-apps.spreadsheet' and trashed = false"
	fileList, err := c.driveSvc.Files.List().
		Q(query).
		Fields("files(id, name, mimeType, modifiedTime)").
		PageSize(50).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to query google drive files: %w", err)
	}

	var items []DriveFileItem
	for _, f := range fileList.Files {
		items = append(items, DriveFileItem{
			ID:           f.Id,
			Name:         f.Name,
			MimeType:     f.MimeType,
			ModifiedTime: f.ModifiedTime,
		})
	}
	return items, nil
}

// GetSpreadsheetInfo 讀取試算表標題與分頁（工作表 Tab）清單。
func (c *Client) GetSpreadsheetInfo(ctx context.Context, inputIDOrURL string, accessToken string) (*SpreadsheetInfo, error) {
	spreadsheetID := app.ExtractSpreadsheetID(inputIDOrURL)
	if spreadsheetID == "" {
		return nil, errors.New("invalid google spreadsheet URL or ID")
	}

	if c.sheetsSvc == nil && strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("google sheets service is not configured and no access token was provided")
	}

	sheetsSvc, err := c.sheetsService(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	resp, err := sheetsSvc.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect spreadsheet %s: %w", spreadsheetID, err)
	}

	var tabs []string
	for _, sheet := range resp.Sheets {
		if sheet.Properties != nil && sheet.Properties.Title != "" {
			tabs = append(tabs, sheet.Properties.Title)
		}
	}

	if len(tabs) == 0 {
		tabs = []string{"表單回覆 1"}
	}

	return &SpreadsheetInfo{
		SpreadsheetID: spreadsheetID,
		Title:         resp.Properties.Title,
		SheetTabs:     tabs,
	}, nil
}

// ReadSheetRows 讀取指定試算表分頁中所有資料列。
func (c *Client) ReadSheetRows(ctx context.Context, inputIDOrURL string, tabName string, accessToken string) ([][]interface{}, error) {
	spreadsheetID := app.ExtractSpreadsheetID(inputIDOrURL)
	if spreadsheetID == "" {
		return nil, errors.New("invalid google spreadsheet URL or ID")
	}

	if tabName == "" {
		tabName = "表單回覆 1"
	}

	if c.sheetsSvc == nil && strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("google sheets service is not configured and no access token was provided")
	}

	readRange := fmt.Sprintf("'%s'!A1:ZZ", tabName)
	sheetsSvc, err := c.sheetsService(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	valueResp, err := sheetsSvc.Spreadsheets.Values.Get(spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read range %s: %w", readRange, err)
	}

	return valueResp.Values, nil
}

func (c *Client) sheetsService(ctx context.Context, accessToken string) (*sheets.Service, error) {
	if strings.TrimSpace(accessToken) == "" {
		return c.sheetsSvc, nil
	}
	return sheets.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})))
}
