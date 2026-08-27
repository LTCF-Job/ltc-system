package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/unicode/norm"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/repository"
)

var (
	reDriverHeader = regexp.MustCompile(`^(.+?)\((\S+)\s+([A-Z][0-9]{9})\)$`)
)

// ImportService 負責批次 Excel/CSV 主檔與個案資料之解析與匯入。
type ImportService struct {
	masterService *MasterService
	siteRepo      *repository.SiteRepository
	vehicleRepo   *repository.VehicleRepository
	driverRepo    *repository.DriverRepository
	caseRepo      *repository.CaseRepository
}

// NewImportService 建立 ImportService 實例。
func NewImportService(
	masterService *MasterService,
	siteRepo *repository.SiteRepository,
	vehicleRepo *repository.VehicleRepository,
	driverRepo *repository.DriverRepository,
	caseRepo *repository.CaseRepository,
) *ImportService {
	return &ImportService{
		masterService: masterService,
		siteRepo:      siteRepo,
		vehicleRepo:   vehicleRepo,
		driverRepo:    driverRepo,
		caseRepo:      caseRepo,
	}
}

// CaseImportErrorItem 代表單筆匯入錯誤明細。
type CaseImportErrorItem struct {
	RowIndex int    `json:"rowIndex"`
	CaseName string `json:"caseName,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// CaseImportWarningItem 代表單筆匯入警告或預設值提醒明細。
type CaseImportWarningItem struct {
	RowIndex int    `json:"rowIndex"`
	CaseName string `json:"caseName,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// WeekdayScheduleDetail 代表個案在單一星期的排班趟數與時段。
type WeekdayScheduleDetail struct {
	Weekday    int16  `json:"weekday"`
	TripCount  int16  `json:"tripCount"`
	DepartTime string `json:"departTime,omitempty"`
	ReturnTime string `json:"returnTime,omitempty"`
}

// CaseImportRowResult 代表個案批次匯入單列解析結果。
type CaseImportRowResult struct {
	RowIndex           int                     `json:"rowIndex"`
	SheetName          string                  `json:"sheetName"`
	Name               string                  `json:"name"`
	NationalID         string                  `json:"nationalId,omitempty"`
	Phone              string                  `json:"phone,omitempty"`
	HomeAddress        string                  `json:"homeAddress,omitempty"`
	Region             string                  `json:"region"`
	ClaimStartDate     string                  `json:"claimStartDate"`
	ServiceCategory    int                     `json:"serviceCategory"`
	ServiceUsageType   int                     `json:"serviceUsageType"`
	SiteName           string                  `json:"siteName"`
	SiteID             *uuid.UUID              `json:"siteId,omitempty"`
	Weekdays           []int16                 `json:"weekdays"`
	WeekdaysText       string                  `json:"weekdaysText,omitempty"`
	WeekdaySchedules   []WeekdayScheduleDetail `json:"weekdaySchedules,omitempty"`
	OutboundTime       string                  `json:"outboundTime,omitempty"`
	InboundTime        string                  `json:"inboundTime,omitempty"`
	OutboundVehicle    string                  `json:"outboundVehicle,omitempty"`
	InboundVehicle     string                  `json:"inboundVehicle,omitempty"`
	TripPattern        int16                   `json:"tripPattern"`
	DistanceKM         float64                 `json:"distanceKm"`
	UnitPrice          float64                 `json:"unitPrice"`
	ServiceDurationMin int16                   `json:"serviceDurationMin"`
	Note               string                  `json:"note,omitempty"`
	IsDraft            bool                    `json:"isDraft"`
	WarningMessage     string                  `json:"warningMessage,omitempty"`
	ErrorMessage       string                  `json:"errorMessage,omitempty"`
}

// CaseImportPreviewResult 批次匯入預覽與統計結構體。
type CaseImportPreviewResult struct {
	TotalRows   int                      `json:"totalRows"`
	ValidRows   int                      `json:"validRows"`
	ErrorRows   int                      `json:"errorRows"`
	WarningRows int                      `json:"warningRows"`
	PreviewRows []map[string]interface{} `json:"previewRows"`
	Errors      []CaseImportErrorItem    `json:"errors"`
	Warnings    []CaseImportWarningItem  `json:"warnings"`
	Rows        []CaseImportRowResult    `json:"rows,omitempty"`
}

// ParseWeekdays 解析「每週據點開放時間」自由文字（規格書 6.2）。
func ParseWeekdays(s string) ([]int16, error) {
	s = norm.NFKC.String(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "周", "週")

	dayMap := map[rune]int16{
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '日': 7, '天': 7,
	}

	rangeSeparators := []string{"到", "~", "-"}
	for _, sep := range rangeSeparators {
		if strings.Contains(s, sep) {
			parts := strings.Split(s, sep)
			if len(parts) == 2 {
				r1 := []rune(parts[0])
				r2 := []rune(parts[1])
				var startDay, endDay int16
				for _, r := range r1 {
					if d, ok := dayMap[r]; ok {
						startDay = d
					}
				}
				for _, r := range r2 {
					if d, ok := dayMap[r]; ok {
						endDay = d
					}
				}
				if startDay > 0 && endDay >= startDay {
					var res []int16
					for d := startDay; d <= endDay; d++ {
						res = append(res, d)
					}
					return res, nil
				}
			}
		}
	}

	var res []int16
	seen := make(map[int16]bool)
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '週' && i+1 < len(runes) {
			if d, ok := dayMap[runes[i+1]]; ok {
				if !seen[d] {
					seen[d] = true
					res = append(res, d)
				}
			}
		}
	}

	if len(res) == 0 {
		return nil, errors.New("cannot parse weekdays from text")
	}

	return res, nil
}

// GenerateCaseImportTemplateCSV 產生個案批次匯入標準 CSV 範本文字。
func GenerateCaseImportTemplateCSV() string {
	var sb strings.Builder
	sb.WriteString("\uFEFF")
	sb.WriteString("個案姓名*,身分證字號*,申報地區*(苗栗/新竹),聯絡電話,住家地址*,開始申報日*(YYYY-MM-DD),服務類別*(1:補助/2:自費),服務使用類型*(1:社區長照/2:社區據點/3:輔具中心/4:身障日照),所屬據點*,單趟里程(公里)*,申報單價(元),服務時長(分鐘),週一趟數(0:不搭/1:單去/2:來回/4:四趟),週一去程時間(HH:mm),週一回程時間(HH:mm),週二趟數(0:不搭/1:單去/2:來回/4:四趟),週二去程時間(HH:mm),週二回程時間(HH:mm),週三趟數(0:不搭/1:單去/2:來回/4:四趟),週三去程時間(HH:mm),週三回程時間(HH:mm),週四趟數(0:不搭/1:單去/2:來回/4:四趟),週四去程時間(HH:mm),週四回程時間(HH:mm),週五趟數(0:不搭/1:單去/2:來回/4:四趟),週五去程時間(HH:mm),週五回程時間(HH:mm),週六趟數(0:不搭/1:單去/2:來回/4:四趟),週六去程時間(HH:mm),週六回程時間(HH:mm),週日趟數(0:不搭/1:單去/2:來回/4:四趟),週日去程時間(HH:mm),週日回程時間(HH:mm),備註\r\n")
	sb.WriteString("張曾阿妹,A202559750,苗栗,0912345678,苗栗縣竹南鎮大營路123號,2026-07-01,1,2,竹南日照據點,5.0,115,10,2,09:00,16:00,2,09:00,16:00,2,09:00,16:00,2,09:00,16:00,2,09:00,16:00,0,,,0,,,週一至週五固定來回\r\n")
	sb.WriteString("李國盛,J123458899,新竹,0922334455,新竹縣竹北市文興路一段200號,2026-07-01,2,1,竹北日照中心,8.0,200,20,2,09:30,15:30,1,09:30,,2,09:30,15:30,0,,,2,09:30,15:30,0,,,0,,,週二僅早上去程\r\n")
	sb.WriteString("王大同,K123456780,苗栗,0933445566,苗栗市中正路50號,2026-07-01,1,4,苗栗復健據點,6.5,115,15,0,,,0,,,0,,,4,08:30,16:30,0,,,0,,,0,,,週四四趟個案\r\n")
	return sb.String()
}

// GenerateCaseImportTemplateExcel 產生個案批次匯入標準 Excel 檔案位元組。
func GenerateCaseImportTemplateExcel() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "個案匯入範本"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{
		"個案姓名*", "身分證字號*", "申報地區*(苗栗/新竹)", "聯絡電話", "住家地址*",
		"開始申報日*(YYYY-MM-DD)", "服務類別*(1:補助/2:自費)", "服務使用類型*(1:社區長照/2:社區據點/3:輔具中心/4:身障日照)", "所屬據點*",
		"單趟里程(公里)*", "申報單價(元)", "服務時長(分鐘)",
		"週一趟數(0:不搭/1:單去/2:來回/4:四趟)", "週一去程時間(HH:mm)", "週一回程時間(HH:mm)",
		"週二趟數(0:不搭/1:單去/2:來回/4:四趟)", "週二去程時間(HH:mm)", "週二回程時間(HH:mm)",
		"週三趟數(0:不搭/1:單去/2:來回/4:四趟)", "週三去程時間(HH:mm)", "週三回程時間(HH:mm)",
		"週四趟數(0:不搭/1:單去/2:來回/4:四趟)", "週四去程時間(HH:mm)", "週四回程時間(HH:mm)",
		"週五趟數(0:不搭/1:單去/2:來回/4:四趟)", "週五去程時間(HH:mm)", "週五回程時間(HH:mm)",
		"週六趟數(0:不搭/1:單去/2:來回/4:四趟)", "週六去程時間(HH:mm)", "週六回程時間(HH:mm)",
		"週日趟數(0:不搭/1:單去/2:來回/4:四趟)", "週日去程時間(HH:mm)", "週日回程時間(HH:mm)",
		"備註",
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2065D1"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheetName, cell, h)
	}
	_ = f.SetRowHeight(sheetName, 1, 32)
	_ = f.SetCellStyle(sheetName, "A1", "AH1", headerStyle)

	sampleRows := [][]interface{}{
		{"張曾阿妹", "A202559750", "苗栗", "0912345678", "苗栗縣竹南鎮大營路123號", "2026-07-01", 1, 2, "竹南日照據點", 5.0, 115, 10, 2, "09:00", "16:00", 2, "09:00", "16:00", 2, "09:00", "16:00", 2, "09:00", "16:00", 2, "09:00", "16:00", 0, "", "", 0, "", "", "週一至週五固定來回"},
		{"李國盛", "J123458899", "新竹", "0922334455", "新竹縣竹北市文興路一段200號", "2026-07-01", 2, 1, "竹北日照中心", 8.0, 200, 20, 2, "09:30", "15:30", 1, "09:30", "", 2, "09:30", "15:30", 0, "", "", 2, "09:30", "15:30", 0, "", "", 0, "", "", "週二僅早上去程"},
		{"王大同", "K123456780", "苗栗", "0933445566", "苗栗市中正路50號", "2026-07-01", 1, 4, "苗栗復健據點", 6.5, 115, 15, 0, "", "", 0, "", "", 0, "", "", 4, "08:30", "16:30", 0, "", "", 0, "", "", 0, "", "", "週四四趟個案"},
	}

	for rIdx, rData := range sampleRows {
		rowNum := rIdx + 2
		for cIdx, val := range rData {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowNum)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}
	_ = f.SetSheetDimension(sheetName, fmt.Sprintf("A1:AH%d", len(sampleRows)+1))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}

// ParseCasesFromExcel 保留相容介面，實作通用串流解析。
func (s *ImportService) ParseCasesFromExcel(r io.Reader) (*CaseImportPreviewResult, error) {
	return s.ParseCases(r, "upload.xlsx")
}

// ParseCases 支援解析 .xlsx, .xls 與 .csv 檔案。
func (s *ImportService) ParseCases(r io.Reader, fileName string) (*CaseImportPreviewResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// 檢查是否為 Excel ZIP 格式 (Magic Number: PK\x03\x04)
	isExcel := len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04
	if isExcel || strings.HasSuffix(strings.ToLower(fileName), ".xlsx") || strings.HasSuffix(strings.ToLower(fileName), ".xls") {
		return s.parseExcelBytes(data)
	}

	return s.parseCSVBytes(data)
}

func (s *ImportService) parseExcelBytes(data []byte) (*CaseImportPreviewResult, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("開啟 Excel 檔案失敗: %w", err)
	}
	defer f.Close()

	var allRawTables [][][]string
	var sheetNames []string
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err == nil && len(rows) > 0 {
			allRawTables = append(allRawTables, rows)
			sheetNames = append(sheetNames, sheet)
		}
	}

	if len(allRawTables) == 0 {
		return nil, errors.New("excel 檔案中無工作表資料")
	}

	return s.processRawTables(allRawTables, sheetNames)
}

func (s *ImportService) parseCSVBytes(data []byte) (*CaseImportPreviewResult, error) {
	// 移除 UTF-8 BOM
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv 解析失敗: %w", err)
	}

	if len(rows) == 0 {
		return nil, errors.New("csv 檔案內容為空")
	}

	return s.processRawTables([][][]string{rows}, []string{"CSV"})
}

func (s *ImportService) processRawTables(tables [][][]string, sheetNames []string) (*CaseImportPreviewResult, error) {
	var results []CaseImportRowResult
	var errorsList []CaseImportErrorItem
	var warningsList []CaseImportWarningItem
	var previewRows []map[string]interface{}

	totalRows := 0
	validRows := 0
	errorRows := 0
	warningRows := 0

	for tableIdx, rows := range tables {
		sheetName := sheetNames[tableIdx]
		if len(rows) < 1 {
			continue
		}

		// 尋找標題列（第 1 列或第 2 列）
		headerRowIdx := 0
		colMap := make(map[string]int)

		for r := 0; r < min(3, len(rows)); r++ {
			rowText := strings.Join(rows[r], ",")
			if strings.Contains(rowText, "姓名") || strings.Contains(rowText, "個案姓名") {
				headerRowIdx = r
				for c, colName := range rows[r] {
					cleanName := strings.TrimSpace(strings.ReplaceAll(colName, "*", ""))
					cleanName = strings.Split(cleanName, "(")[0]
					cleanName = strings.Split(cleanName, "（")[0]
					cleanName = strings.TrimSpace(cleanName)
					if cleanName != "" {
						colMap[cleanName] = c
					}
				}
				break
			}
		}

		for rIdx := headerRowIdx + 1; rIdx < len(rows); rIdx++ {
			row := rows[rIdx]
			if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
				continue
			}

			getVal := func(keys ...string) string {
				for _, k := range keys {
					if idx, ok := colMap[k]; ok && idx < len(row) {
						v := strings.TrimSpace(row[idx])
						if v != "" {
							return v
						}
					}
					for colName, idx := range colMap {
						if (strings.Contains(colName, k) || strings.Contains(k, colName)) && idx < len(row) {
							v := strings.TrimSpace(row[idx])
							if v != "" {
								return v
							}
						}
					}
				}
				return ""
			}

			// 依固定 index 降級取得（相容舊版無標題或特定格式）
			getIdx := func(idx int) string {
				if idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}

			name := getVal("個案姓名", "姓名")
			if name == "" && len(colMap) == 0 {
				name = getIdx(1)
				if name == "" {
					name = getIdx(0)
				}
			}

			if name == "" || strings.HasPrefix(name, "例:") || strings.HasPrefix(name, "例：") {
				continue
			}

			totalRows++
			actualRowIndex := rIdx + 1

			nationalID := getVal("身分證字號", "身分證")
			regionStr := getVal("申報地區", "地區", "區域")
			phone := getVal("聯絡電話", "電話")
			homeAddress := getVal("住家地址", "地址")
			claimStartDate := getVal("開始申報日", "幾號開始申報", "申報開始日")
			siteName := getVal("所屬據點", "據點", "據點名稱")
			distanceStr := getVal("單趟里程(公里)", "單趟里程", "里程", "里程數")
			unitPriceStr := getVal("申報單價(元)", "申報單價", "單價")
			durationStr := getVal("服務時長(分鐘)", "服務時長", "時長")
			serviceCatStr := getVal("服務類別")
			serviceUsageStr := getVal("服務使用類型")
			note := getVal("備註")

			// 相容舊版欄位順序 (序號|姓名|地區|幾號開始申報|據點|每週開放時間|去程時間|回程時間...)
			if regionStr == "" && len(colMap) == 0 {
				regionStr = getIdx(2)
			}
			if claimStartDate == "" && len(colMap) == 0 {
				claimStartDate = getIdx(3)
			}
			if siteName == "" && len(colMap) == 0 {
				siteName = getIdx(4)
			}

			region := "miaoli"
			if strings.Contains(regionStr, "新竹") || strings.ToLower(regionStr) == "hsinchu" {
				region = "hsinchu"
			}

			if claimStartDate == "" {
				claimStartDate = time.Now().Format("2006-01-02")
			}

			unitPrice := 115.0
			if unitPriceStr != "" {
				if val, err := strconv.ParseFloat(unitPriceStr, 64); err == nil && val > 0 {
					unitPrice = val
				}
			}

			durationMin := int16(10)
			if durationStr != "" {
				if val, err := strconv.Atoi(durationStr); err == nil && val > 0 {
					durationMin = int16(val)
				}
			}

			distanceKM := 0.0
			if distanceStr != "" {
				if val, err := strconv.ParseFloat(distanceStr, 64); err == nil {
					distanceKM = val
				}
			}

			serviceCategory := 1
			if serviceCatStr != "" {
				if strings.Contains(serviceCatStr, "2") || strings.Contains(serviceCatStr, "自費") {
					serviceCategory = 2
				}
			}

			serviceUsageType := 2
			if serviceUsageStr != "" {
				if strings.Contains(serviceUsageStr, "1") {
					serviceUsageType = 1
				} else if strings.Contains(serviceUsageStr, "3") {
					serviceUsageType = 3
				} else if strings.Contains(serviceUsageStr, "4") {
					serviceUsageType = 4
				}
			}

			// 解析週一至週日各別排班與時段
			dayNames := []string{"週一", "週二", "週三", "週四", "週五", "週六", "週日"}
			var weekdayScheds []WeekdayScheduleDetail
			var activeWeekdays []int16
			var maxTripPattern int16 = 2
			hasDailyCols := false

			for dayIdx, dName := range dayNames {
				wDay := int16(dayIdx + 1)
				tripCountStr := getVal(dName + "趟數")
				outTime := getVal(dName + "去程時間")
				inTime := getVal(dName + "回程時間")

				tripCount := int16(0)
				if tripCountStr != "" {
					hasDailyCols = true
					if val, err := strconv.Atoi(tripCountStr); err == nil {
						tripCount = int16(val)
					} else if strings.Contains(tripCountStr, "單") {
						tripCount = 1
					} else if strings.Contains(tripCountStr, "來回") || strings.Contains(tripCountStr, "2") {
						tripCount = 2
					} else if strings.Contains(tripCountStr, "四") || strings.Contains(tripCountStr, "4") {
						tripCount = 4
					}
				} else if outTime != "" || inTime != "" {
					hasDailyCols = true
					if outTime != "" && inTime != "" {
						tripCount = 2
					} else {
						tripCount = 1
					}
				}

				if tripCount > 0 {
					activeWeekdays = append(activeWeekdays, wDay)
					if tripCount > maxTripPattern {
						maxTripPattern = tripCount
					}
					weekdayScheds = append(weekdayScheds, WeekdayScheduleDetail{
						Weekday:    wDay,
						TripCount:  tripCount,
						DepartTime: outTime,
						ReturnTime: inTime,
					})
				}
			}

			// 若未填寫各星期欄位，相容通用排班欄位
			outboundTime := getVal("去程時間")
			inboundTime := getVal("回程時間")
			weekdaysText := getVal("每週搭乘日", "每週據點開放時間", "開放時間")

			if !hasDailyCols {
				if weekdaysText != "" {
					if wds, err := ParseWeekdays(weekdaysText); err == nil {
						activeWeekdays = wds
					}
				}
				if len(activeWeekdays) == 0 {
					activeWeekdays = []int16{1, 2, 3, 4, 5}
				}

				tripPatternStr := getVal("趟數型態", "趟數")
				if tripPatternStr != "" {
					if val, err := strconv.Atoi(tripPatternStr); err == nil && (val == 1 || val == 2 || val == 4) {
						maxTripPattern = int16(val)
					}
				} else {
					if outboundTime != "" && inboundTime != "" {
						maxTripPattern = 2
					} else if outboundTime != "" || inboundTime != "" {
						maxTripPattern = 1
					}
				}
			}

			if outboundTime == "" && len(weekdayScheds) > 0 {
				outboundTime = weekdayScheds[0].DepartTime
			}
			if inboundTime == "" && len(weekdayScheds) > 0 {
				inboundTime = weekdayScheds[0].ReturnTime
			}
			if outboundTime == "" {
				outboundTime = "09:00"
			}
			if inboundTime == "" {
				inboundTime = "16:00"
			}

			rowRes := CaseImportRowResult{
				RowIndex:           actualRowIndex,
				SheetName:          sheetName,
				Name:               name,
				NationalID:         nationalID,
				Phone:              phone,
				HomeAddress:        homeAddress,
				Region:             region,
				ClaimStartDate:     claimStartDate,
				ServiceCategory:    serviceCategory,
				ServiceUsageType:   serviceUsageType,
				SiteName:           siteName,
				Weekdays:           activeWeekdays,
				WeekdaySchedules:   weekdayScheds,
				OutboundTime:       outboundTime,
				InboundTime:        inboundTime,
				TripPattern:        maxTripPattern,
				DistanceKM:         distanceKM,
				UnitPrice:          unitPrice,
				ServiceDurationMin: durationMin,
				Note:               note,
				IsDraft:            nationalID == "" || homeAddress == "",
			}

			hasError := false
			hasWarning := false

			// 里程數檢核
			if distanceKM <= 0 {
				rowRes.DistanceKM = 5.0
				rowRes.WarningMessage += "未填寫單趟里程，已套用預設值 5.0 公里; "
				warningsList = append(warningsList, CaseImportWarningItem{
					RowIndex: actualRowIndex,
					CaseName: name,
					Field:    "單趟里程",
					Message:  "單趟里程未填或為 0，已預設帶入 5.0 公里",
				})
				hasWarning = true
			}

			if rowRes.IsDraft {
				rowRes.WarningMessage += "缺少身分證或住家地址，將建立為草稿個案; "
				warningsList = append(warningsList, CaseImportWarningItem{
					RowIndex: actualRowIndex,
					CaseName: name,
					Field:    "個案基本資料",
					Message:  "缺少身分證字號或住家地址，匯入後為草稿狀態",
				})
				hasWarning = true
			}

			if hasError {
				errorRows++
			} else {
				validRows++
				if hasWarning {
					warningRows++
				}
			}

			results = append(results, rowRes)

			// 組合前端預覽表格用資料
			var wText string
			if len(weekdayScheds) > 0 {
				var parts []string
				for _, ws := range weekdayScheds {
					parts = append(parts, fmt.Sprintf("週%s(%d趟)", dayNames[ws.Weekday-1][3:], ws.TripCount))
				}
				wText = strings.Join(parts, ", ")
			} else {
				wText = fmt.Sprintf("每週 %d 天", len(activeWeekdays))
			}

			previewRows = append(previewRows, map[string]interface{}{
				"rowIndex":       actualRowIndex,
				"name":           name,
				"nationalId":     crypto.Mask(nationalID),
				"region":         region,
				"claimStartDate": claimStartDate,
				"siteName":       siteName,
				"weekdays":       wText,
				"departTime":     outboundTime,
				"returnTime":     inboundTime,
				"tripPattern":    maxTripPattern,
				"distanceKm":     rowRes.DistanceKM,
				"unitPrice":      unitPrice,
				"isDraft":        rowRes.IsDraft,
				"__hasError":     hasError,
				"__hasWarning":   hasWarning,
			})
		}
	}

	return &CaseImportPreviewResult{
		TotalRows:   totalRows,
		ValidRows:   validRows,
		ErrorRows:   errorRows,
		WarningRows: warningRows,
		PreviewRows: previewRows,
		Errors:      errorsList,
		Warnings:    warningsList,
		Rows:        results,
	}, nil
}

// CommitCases 將通過檢核的個案資料正式寫入資料庫。
func (s *ImportService) CommitCases(ctx context.Context, preview *CaseImportPreviewResult, actorID uuid.UUID, actorRole, ip, ua string) (int, error) {
	if preview == nil || len(preview.Rows) == 0 {
		return 0, nil
	}

	createdCount := 0
	for _, row := range preview.Rows {
		if row.ErrorMessage != "" {
			continue
		}

		claimStart, err := time.Parse("2006-01-02", row.ClaimStartDate)
		if err != nil {
			claimStart = time.Now()
		}

		natID := row.NationalID
		if natID == "" {
			natID = fmt.Sprintf("A%09d", time.Now().UnixNano()%1000000000)
		}
		addr := row.HomeAddress
		if addr == "" {
			addr = "待補住家地址"
		}

		status := "active"
		if row.IsDraft {
			status = "suspended"
		}

		caseReq := CreateCaseRequest{
			Name:             row.Name,
			NationalID:       natID,
			HomeAddress:      addr,
			Region:           row.Region,
			ClaimStartDate:   claimStart,
			ServiceCategory:  row.ServiceCategory,
			ServiceUsageType: row.ServiceUsageType,
			Status:           status,
		}

		caseEntity, err := s.masterService.CreateCase(ctx, caseReq, actorID, actorRole, ip, ua)
		if err != nil {
			// 若身分證重複則略過
			continue
		}

		// 關聯據點與建立排班
		var siteID uuid.UUID
		if s.siteRepo != nil {
			siteList, _, _ := s.siteRepo.List(ctx, row.Region, "", 1, 100)
			for _, st := range siteList {
				if st.Name == row.SiteName || strings.Contains(st.Name, row.SiteName) {
					siteID = st.ID
					break
				}
			}
			if siteID == uuid.Nil && len(siteList) > 0 {
				siteID = siteList[0].ID
			}
		}

		if siteID != uuid.Nil && len(row.Weekdays) > 0 {
			var legs []CreateScheduleLegItemRequest
			if row.TripPattern == 1 {
				legs = append(legs, CreateScheduleLegItemRequest{
					LegSeq:     1,
					Direction:  "outbound",
					DepartTime: row.OutboundTime,
				})
			} else if row.TripPattern == 4 {
				legs = append(legs,
					CreateScheduleLegItemRequest{LegSeq: 1, Direction: "outbound", DepartTime: "08:30"},
					CreateScheduleLegItemRequest{LegSeq: 2, Direction: "inbound", DepartTime: "11:30"},
					CreateScheduleLegItemRequest{LegSeq: 3, Direction: "outbound", DepartTime: "13:30"},
					CreateScheduleLegItemRequest{LegSeq: 4, Direction: "inbound", DepartTime: "16:30"},
				)
			} else {
				legs = append(legs,
					CreateScheduleLegItemRequest{LegSeq: 1, Direction: "outbound", DepartTime: row.OutboundTime},
					CreateScheduleLegItemRequest{LegSeq: 2, Direction: "inbound", DepartTime: row.InboundTime},
				)
			}

			schedNote := row.Note
			_, _ = s.masterService.CreateCaseSchedule(ctx, CreateScheduleRequest{
				CaseID:             caseEntity.ID,
				SiteID:             siteID,
				EffectiveFrom:      claimStart,
				Weekdays:           row.Weekdays,
				TripPattern:        row.TripPattern,
				UnitPrice:          row.UnitPrice,
				DistanceKM:         row.DistanceKM,
				ServiceDurationMin: row.ServiceDurationMin,
				ServiceCode:        "BD03",
				Note:               &schedNote,
				Legs:               legs,
			})
		}

		createdCount++
	}

	return createdCount, nil
}
