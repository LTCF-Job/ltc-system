package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/unicode/norm"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/repository"
)

var (
	reDriverHeader = regexp.MustCompile(`^(.+?)\((\S+)\s+([A-Z][0-9]{9})\)$`)
)

// ImportService 負責批次 Excel 主檔與個案資料之解析與匯入。
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

// CaseImportRowResult 代表個案批次匯入單列解析結果。
type CaseImportRowResult struct {
	RowIndex           int       `json:"rowIndex"`
	SheetName          string    `json:"sheetName"`
	Name               string    `json:"name"`
	Region             string    `json:"region"`
	ClaimStartDate     string    `json:"claimStartDate"`
	SiteName           string    `json:"siteName"`
	SiteID             *uuid.UUID `json:"siteId,omitempty"`
	Weekdays           []int16   `json:"weekdays"`
	OutboundTime       string    `json:"outboundTime,omitempty"`
	InboundTime        string    `json:"inboundTime,omitempty"`
	OutboundVehicle    string    `json:"outboundVehicle,omitempty"`
	InboundVehicle     string    `json:"inboundVehicle,omitempty"`
	TripPattern        int16     `json:"tripPattern"`
	DistanceKM         float64   `json:"distanceKm"`
	UnitPrice          float64   `json:"unitPrice"`
	ServiceDurationMin int16     `json:"serviceDurationMin"`
	IsDraft            bool      `json:"isDraft"`
	WarningMessage     string    `json:"warningMessage,omitempty"`
	ErrorMessage       string    `json:"errorMessage,omitempty"`
}

// CaseImportPreviewResult 批次匯入預覽與統計結構體。
type CaseImportPreviewResult struct {
	TotalRows int                   `json:"totalRows"`
	ValidRows int                   `json:"validRows"`
	ErrorRows int                   `json:"errorRows"`
	Rows      []CaseImportRowResult `json:"rows"`
}

// ParseWeekdays 解析「每週據點開放時間」自由文字（規格書 6.2）。
func ParseWeekdays(s string) ([]int16, error) {
	s = norm.NFKC.String(s)
	// 移除空格
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\t", "")
	// 周統一轉週
	s = strings.ReplaceAll(s, "周", "週")

	dayMap := map[rune]int16{
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '日': 7, '天': 7,
	}

	// 檢查區間形式（例如「週一到週五」、「週一~週五」、「週一-週五」）
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

	// 逐字萃取週幾
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

// ParseCasesFromExcel 解析個案新增資料.xlsx 之全部內容。
func (s *ImportService) ParseCasesFromExcel(r io.Reader) (*CaseImportPreviewResult, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, fmt.Errorf("failed to read excel stream: %w", err)
	}

	f, err := excelize.OpenReader(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	var allResults []CaseImportRowResult
	totalRows := 0
	validRows := 0
	errorRows := 0

	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil || len(rows) < 3 {
			continue // 略過無資料或格式不符的工作表
		}

		// 第 1 列為大標題，第 2 列為標題列，第 3 列起為資料列
		for rowIdx := 2; rowIdx < len(rows); rowIdx++ {
			row := rows[rowIdx]
			if len(row) < 2 {
				continue
			}

			name := strings.TrimSpace(row[1])
			if name == "" || strings.HasPrefix(name, "例:") || strings.HasPrefix(name, "例：") {
				// 略過範例列或空列
				continue
			}

			totalRows++
			res := CaseImportRowResult{
				RowIndex:           rowIdx + 1,
				SheetName:          sheetName,
				Name:               name,
				UnitPrice:          115.00,
				ServiceDurationMin: 10,
				IsDraft:            true, // 預設缺少身分證地址為草稿
			}

			// 申報地區
			if len(row) > 2 {
				reg := strings.TrimSpace(row[2])
				if strings.Contains(reg, "苗栗") {
					res.Region = "miaoli"
				} else if strings.Contains(reg, "新竹") {
					res.Region = "hsinchu"
				} else {
					res.Region = reg
				}
			}

			// 幾號開始申報
			if len(row) > 3 {
				dateStr := strings.TrimSpace(row[3])
				res.ClaimStartDate = dateStr
			}

			// 據點名稱
			if len(row) > 4 {
				res.SiteName = strings.TrimSpace(row[4])
			}

			// 開放時間解析
			if len(row) > 5 {
				weekdaysText := strings.TrimSpace(row[5])
				wds, err := ParseWeekdays(weekdaysText)
				if err != nil {
					res.WarningMessage += "每週開放時間解析失敗，需人工確認; "
				} else {
					res.Weekdays = wds
				}
			}

			// 去程時間與車名
			if len(row) > 6 {
				res.OutboundTime = strings.TrimSpace(row[6])
			}
			if len(row) > 8 {
				res.OutboundVehicle = strings.TrimSpace(row[8])
			}

			// 回程時間與車名
			if len(row) > 7 {
				res.InboundTime = strings.TrimSpace(row[7])
			}
			if len(row) > 9 {
				res.InboundVehicle = strings.TrimSpace(row[9])
			}

			// 趟數推導
			if res.OutboundTime != "" && res.InboundTime != "" {
				res.TripPattern = 2
			} else if res.OutboundTime != "" || res.InboundTime != "" {
				res.TripPattern = 1
			} else {
				res.TripPattern = 2 // 預設 2 趟
			}

			// 里程數檢查
			if res.DistanceKM <= 0 {
				res.ErrorMessage = "缺少里程數 (distanceKm)"
				errorRows++
			} else {
				validRows++
			}

			allResults = append(allResults, res)
		}
	}

	return &CaseImportPreviewResult{
		TotalRows: totalRows,
		ValidRows: validRows,
		ErrorRows: errorRows,
		Rows:      allResults,
	}, nil
}

// ParseScheduleWorkbook 解析 (參考用)交通車接送班表.xlsx 之據點與司機資訊。
func (s *ImportService) ParseScheduleWorkbook(r io.Reader) (sites []repository.SiteEntity, drivers []repository.DriverEntity, err error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, nil, fmt.Errorf("failed to read stream: %w", err)
	}

	f, err := excelize.OpenReader(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open excel: %w", err)
	}
	defer f.Close()

	// 1. 解析據點工作表
	siteRows, err := f.GetRows("據點")
	if err == nil && len(siteRows) > 1 {
		for i := 1; i < len(siteRows); i++ {
			row := siteRows[i]
			if len(row) < 3 || strings.TrimSpace(row[0]) == "" {
				continue
			}
			siteName := strings.TrimSpace(row[1])
			siteAddress := strings.TrimSpace(row[2])
			if siteName == "" || siteAddress == "" {
				continue
			}

			sites = append(sites, repository.SiteEntity{
				ID:       uuid.New(),
				Code:     fmt.Sprintf("S%03d", len(sites)+1),
				Name:     siteName,
				Address:  siteAddress,
				Region:   "hsinchu",
				OpenDays: []int16{1, 2, 3, 4, 5},
				Status:   "active",
			})
		}
	}

	// 2. 解析司機 (從 11507接送時間表 或 班表 A 欄)
	driverRows, err := f.GetRows("11507接送時間表")
	if err == nil {
		seenNID := make(map[string]bool)
		for _, row := range driverRows {
			if len(row) == 0 {
				continue
			}
			cellText := strings.TrimSpace(row[0])
			matches := reDriverHeader.FindStringSubmatch(cellText)
			if len(matches) == 4 {
				name := strings.TrimSpace(matches[2])
				nid := strings.TrimSpace(matches[3])
				if !crypto.ValidateNationalID(nid) || seenNID[nid] {
					continue
				}
				seenNID[nid] = true

				drivers = append(drivers, repository.DriverEntity{
					ID:               uuid.New(),
					Code:             fmt.Sprintf("D%03d", len(drivers)+1),
					Name:             name,
					NameNormalized:   namenorm.Normalize(name),
					NationalIDMasked: crypto.Mask(nid),
					Region:           "hsinchu",
					Status:           "active",
				})
			}
		}
	}

	return sites, drivers, nil
}
