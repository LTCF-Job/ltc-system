package google

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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
	isOffline bool
}

// NewClient 建立 Google API Client；若 saJSON 為空則進入離線/示範模式。
func NewClient(ctx context.Context, saJSON string) (*Client, error) {
	if strings.TrimSpace(saJSON) == "" {
		return &Client{isOffline: true}, nil
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
		isOffline: false,
	}, nil
}

// ExtractSpreadsheetID 從 Google 試算表完整 URL 或 ID 中擷取純 spreadsheetId。
func ExtractSpreadsheetID(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// 匹配 https://docs.google.com/spreadsheets/d/{ID}/...
	re := regexp.MustCompile(`/spreadsheets/d/([a-zA-Z0-9-_]+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}

	// 若非網址格式且不含斜線，視為直接傳入 ID
	if !strings.Contains(input, "/") && !strings.Contains(input, "?") {
		return input
	}

	return input
}

// ListDriveSheets 查詢 Service Account 可存取之試算表清單。
func (c *Client) ListDriveSheets(ctx context.Context) ([]DriveFileItem, error) {
	if c.isOffline || c.driveSvc == nil {
		// 離線降級示範資料
		return []DriveFileItem{
			{
				ID:           "1A2B3C4D5E6F7G8H9I0J_zhubei1",
				Name:         "竹北一車每日接送回報 (回覆)",
				MimeType:     "application/vnd.google-apps.spreadsheet",
				ModifiedTime: "2026-08-27 10:00:00",
			},
			{
				ID:           "1A2B3C4D5E6F7G8H9I0J_zhubei2",
				Name:         "竹北二車每日接送回報 (回覆)",
				MimeType:     "application/vnd.google-apps.spreadsheet",
				ModifiedTime: "2026-08-27 09:30:00",
			},
			{
				ID:           "1A2B3C4D5E6F7G8H9I0J_zhunan1",
				Name:         "竹南1車每日接送回報 (回覆)",
				MimeType:     "application/vnd.google-apps.spreadsheet",
				ModifiedTime: "2026-08-26 18:20:00",
			},
			{
				ID:           "1A2B3C4D5E6F7G8H9I0J_zhunan2",
				Name:         "竹南2車每日接送回報 (回覆)",
				MimeType:     "application/vnd.google-apps.spreadsheet",
				ModifiedTime: "2026-08-26 17:45:00",
			},
		}, nil
	}

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
	spreadsheetID := ExtractSpreadsheetID(inputIDOrURL)
	if spreadsheetID == "" {
		return nil, errors.New("invalid google spreadsheet URL or ID")
	}

	if (c.isOffline || c.sheetsSvc == nil) && strings.TrimSpace(accessToken) == "" {
		// 離線降級示範結構
		return &SpreadsheetInfo{
			SpreadsheetID: spreadsheetID,
			Title:         "竹北一車每日接送回報 (回覆)",
			SheetTabs:     []string{"8月回報", "7月回報", "表單回覆 1"},
		}, nil
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
	spreadsheetID := ExtractSpreadsheetID(inputIDOrURL)
	if spreadsheetID == "" {
		return nil, errors.New("invalid google spreadsheet URL or ID")
	}

	if tabName == "" {
		tabName = "表單回覆 1"
	}

	if (c.isOffline || c.sheetsSvc == nil) && strings.TrimSpace(accessToken) == "" {
		// 離線示範標題與資料列
		return [][]interface{}{
			{"時間戳記", "今天日期", "今日駕駛人", "蔡曾切（去）", "蔡曾切（回）", "問題回報"},
			{"2026/08/25 下午 3:30:00", "2026-08-25", "陳司機", "有坐", "有坐", "無"},
			{"2026/08/26 下午 3:30:00", "2026-08-26", "陳司機", "有坐", "沒坐", "無"},
		}, nil
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
