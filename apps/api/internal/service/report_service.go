package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"
)

// TripSummaryCaseRow 代表單一個案之趟數統計。
type TripSummaryCaseRow struct {
	CaseID        string `json:"caseId"`
	CaseCode      string `json:"caseCode"`
	CaseName      string `json:"caseName"`
	OutboundCount int    `json:"outboundCount"`
	InboundCount  int    `json:"inboundCount"`
	TotalCount    int    `json:"totalCount"`
}

// TripSummaryVehicle 代表單一車輛之趟數分組。
type TripSummaryVehicle struct {
	VehicleID        string               `json:"vehicleId"`
	VehicleName      string               `json:"vehicleName"`
	PlateNo          string               `json:"plateNo"`
	DriverName       *string              `json:"driverName,omitempty"`
	Rows             []TripSummaryCaseRow `json:"rows"`
	SubtotalOutbound int                  `json:"subtotalOutbound"`
	SubtotalInbound  int                  `json:"subtotalInbound"`
	SubtotalTotal    int                  `json:"subtotalTotal"`
}

// TripSummaryReport 代表車輛趟數表報表總體結構。
type TripSummaryReport struct {
	PeriodYM           string               `json:"periodYm"`
	Region             *string              `json:"region,omitempty"`
	GeneratedAt        string               `json:"generatedAt"`
	Vehicles           []TripSummaryVehicle `json:"vehicles"`
	GrandTotalOutbound int                  `json:"grandTotalOutbound"`
	GrandTotalInbound  int                  `json:"grandTotalInbound"`
	GrandTotal         int                  `json:"grandTotal"`
}

// HsinchuScheduleItem 代表新竹接送時刻表單一排班項目。
type HsinchuScheduleItem struct {
	Direction   string  `json:"direction"` // outbound, inbound
	RunNo       int16   `json:"runNo"`     // 趟次 (1, 2, 3...)
	CaseCode    string  `json:"caseCode"`
	CaseName    string  `json:"caseName"`
	Note        *string `json:"note,omitempty"`
	DepartTime  string  `json:"departTime"`
	Origin      string  `json:"origin"`
	ArriveTime  *string `json:"arriveTime,omitempty"`
	Destination string  `json:"destination"`
	VehicleName string  `json:"vehicleName"`
	SiteName    string  `json:"siteName"`
}

// HsinchuScheduleReport 代表新竹接送時刻表完整結構。
type HsinchuScheduleReport struct {
	GeneratedAt string                `json:"generatedAt"`
	SiteName    *string               `json:"siteName,omitempty"`
	VehicleName *string               `json:"vehicleName,omitempty"`
	Outbound    []HsinchuScheduleItem `json:"outbound"`
	Inbound     []HsinchuScheduleItem `json:"inbound"`
}

// ReportService 提供報表資料彙總與 Excel 檔案產生服務。
type ReportService struct {
	db *pgxpool.Pool
}

// NewReportService 建立 ReportService 實例。
func NewReportService(db *pgxpool.Pool) *ReportService {
	return &ReportService{db: db}
}

// GetTripSummary 查詢車輛趟數表結構化資料。
func (s *ReportService) GetTripSummary(ctx context.Context, periodYm string, region *string, vehicleID *uuid.UUID) (*TripSummaryReport, error) {
	report := &TripSummaryReport{
		PeriodYM:    periodYm,
		Region:      region,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Vehicles:    []TripSummaryVehicle{},
	}

	if s.db == nil {
		return report, nil
	}

	var startDate, endDate time.Time
	if strings.Contains(periodYm, "-") {
		parts := strings.Split(periodYm, "-")
		if len(parts) == 2 {
			var year, month int
			fmt.Sscanf(parts[0], "%d", &year)
			fmt.Sscanf(parts[1], "%d", &month)
			if year < 1000 {
				year += 1911 // 民國轉西元
			}
			startDate = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			endDate = startDate.AddDate(0, 1, 0)
		}
	} else if len(periodYm) == 5 {
		var rocYear, month int
		fmt.Sscanf(periodYm[:3], "%d", &rocYear)
		fmt.Sscanf(periodYm[3:], "%d", &month)
		startDate = time.Date(rocYear+1911, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0)
	}

	if startDate.IsZero() {
		startDate = time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0)
	}

	vehQuery := "SELECT id, plate_no, display_name, region FROM vehicles WHERE 1=1"
	args := []interface{}{}
	argIdx := 1
	if region != nil && *region != "" {
		vehQuery += fmt.Sprintf(" AND region = $%d", argIdx)
		args = append(args, *region)
		argIdx++
	}
	if vehicleID != nil {
		vehQuery += fmt.Sprintf(" AND id = $%d", argIdx)
		args = append(args, *vehicleID)
		argIdx++
	}
	vehQuery += " ORDER BY display_name ASC"

	vehRows, err := s.db.Query(ctx, vehQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicles for report: %w", err)
	}
	defer vehRows.Close()

	type vehItem struct {
		id          uuid.UUID
		plateNo     string
		displayName string
		region      string
	}
	var vehicles []vehItem
	for vehRows.Next() {
		var v vehItem
		if err := vehRows.Scan(&v.id, &v.plateNo, &v.displayName, &v.region); err == nil {
			vehicles = append(vehicles, v)
		}
	}

	for _, v := range vehicles {
		vehSummary := TripSummaryVehicle{
			VehicleID:   v.id.String(),
			VehicleName: v.displayName,
			PlateNo:     v.plateNo,
			Rows:        []TripSummaryCaseRow{},
		}

		statQuery := `
			SELECT 
				c.id, c.code, c.name,
				COALESCE(SUM(CASE WHEN r.leg_seq IN (1, 3) THEN 1 ELSE 0 END), 0) AS outbound_count,
				COALESCE(SUM(CASE WHEN r.leg_seq IN (2, 4) THEN 1 ELSE 0 END), 0) AS inbound_count,
				COUNT(r.id) AS total_count
			FROM cases c
			JOIN ride_records r ON r.case_id = c.id
			WHERE r.vehicle_id = $1
			  AND r.service_date >= $2 AND r.service_date < $3
			  AND r.effective_status = 'boarded'
			GROUP BY c.id, c.code, c.name
			ORDER BY c.code ASC
		`

		rRows, err := s.db.Query(ctx, statQuery, v.id, startDate, endDate)
		if err != nil {
			continue
		}

		for rRows.Next() {
			var caseID uuid.UUID
			var code, name string
			var outbound, inbound, total int
			if err := rRows.Scan(&caseID, &code, &name, &outbound, &inbound, &total); err == nil {
				vehSummary.Rows = append(vehSummary.Rows, TripSummaryCaseRow{
					CaseID:        caseID.String(),
					CaseCode:      code,
					CaseName:      name,
					OutboundCount: outbound,
					InboundCount:  inbound,
					TotalCount:    total,
				})
				vehSummary.SubtotalOutbound += outbound
				vehSummary.SubtotalInbound += inbound
				vehSummary.SubtotalTotal += total
			}
		}
		rRows.Close()

		if len(vehSummary.Rows) > 0 {
			report.Vehicles = append(report.Vehicles, vehSummary)
			report.GrandTotalOutbound += vehSummary.SubtotalOutbound
			report.GrandTotalInbound += vehSummary.SubtotalInbound
			report.GrandTotal += vehSummary.SubtotalTotal
		}
	}

	return report, nil
}

// GenerateTripSummaryExcel 產生車輛趟數表 Excel 檔案。
func (s *ReportService) GenerateTripSummaryExcel(ctx context.Context, periodYm string, region *string, vehicleID *uuid.UUID) ([]byte, error) {
	report, err := s.GetTripSummary(ctx, periodYm, region, vehicleID)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := "Sheet1"
	isFirst := true

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1D5B79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	subtotalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E1EFF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})

	if len(report.Vehicles) == 0 {
		f.SetCellValue(defaultSheet, "A1", "此月份與條件下無搭乘紀錄")
	}

	for _, v := range report.Vehicles {
		sheetName := v.VehicleName
		if isFirst {
			f.SetSheetName(defaultSheet, sheetName)
			isFirst = false
		} else {
			f.NewSheet(sheetName)
		}

		// 表頭
		f.SetCellValue(sheetName, "A1", fmt.Sprintf("長照交通接送 車輛趟數表 (%s)", report.PeriodYM))
		f.SetCellValue(sheetName, "A2", fmt.Sprintf("車輛名稱：%s (%s)", v.VehicleName, v.PlateNo))

		headers := []string{"個案編號", "個案姓名", "去程趟數", "回程趟數", "個人合計"}
		for colIdx, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, 4)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		rowNum := 5
		for _, r := range v.Rows {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), r.CaseCode)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), r.CaseName)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), r.OutboundCount)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), r.InboundCount)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), r.TotalCount)
			rowNum++
		}

		// 車輛小計列
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), "車輛小計")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), "")
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), v.SubtotalOutbound)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), v.SubtotalInbound)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), v.SubtotalTotal)
		for c := 1; c <= 5; c++ {
			cell, _ := excelize.CoordinatesToCellName(c, rowNum)
			f.SetCellStyle(sheetName, cell, cell, subtotalStyle)
		}

		f.SetColWidth(sheetName, "A", "A", 16)
		f.SetColWidth(sheetName, "B", "B", 18)
		f.SetColWidth(sheetName, "C", "E", 14)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write excel buffer: %w", err)
	}

	return buf.Bytes(), nil
}

// GetHsinchuSchedule 查詢新竹接送時刻表排班結構。
func (s *ReportService) GetHsinchuSchedule(ctx context.Context, siteID *uuid.UUID, vehicleID *uuid.UUID) (*HsinchuScheduleReport, error) {
	report := &HsinchuScheduleReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Outbound:    []HsinchuScheduleItem{},
		Inbound:     []HsinchuScheduleItem{},
	}

	if s.db == nil {
		return report, nil
	}

	query := `
		SELECT 
			l.direction, l.run_no, c.code, c.name, cs.note,
			to_char(l.depart_time, 'HH24:MI') as depart_time,
			to_char(l.arrive_time, 'HH24:MI') as arrive_time,
			c.home_address, s.address as site_address,
			COALESCE(v.display_name, '') as vehicle_name,
			s.name as site_name
		FROM schedule_legs l
		JOIN case_schedules cs ON cs.id = l.schedule_id
		JOIN cases c ON c.id = cs.case_id
		JOIN sites s ON s.id = cs.site_id
		LEFT JOIN vehicles v ON v.id = l.vehicle_id
		WHERE c.region = 'hsinchu'
		  AND c.status = 'active'
	`
	args := []interface{}{}
	argIdx := 1

	if siteID != nil {
		query += fmt.Sprintf(" AND s.id = $%d", argIdx)
		args = append(args, *siteID)
		argIdx++
	}
	if vehicleID != nil {
		query += fmt.Sprintf(" AND l.vehicle_id = $%d", argIdx)
		args = append(args, *vehicleID)
		argIdx++
	}

	// 排序規則：先去程再去回程，再依趟序 run_no 與出發時間 depart_time 排序
	query += " ORDER BY l.direction DESC, l.run_no ASC, l.depart_time ASC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hsinchu schedule: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var direction string
		var runNo int16
		var caseCode, caseName string
		var note *string
		var departTime string
		var arriveTime *string
		var homeAddress, siteAddress string
		var vehicleName, siteName string

		if err := rows.Scan(
			&direction, &runNo, &caseCode, &caseName, &note,
			&departTime, &arriveTime, &homeAddress, &siteAddress,
			&vehicleName, &siteName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule item: %w", err)
		}

		var origin, destination string
		if direction == "outbound" {
			origin = homeAddress
			destination = siteAddress
		} else {
			origin = siteAddress
			destination = homeAddress
		}

		item := HsinchuScheduleItem{
			Direction:   direction,
			RunNo:       runNo,
			CaseCode:    caseCode,
			CaseName:    caseName,
			Note:        note,
			DepartTime:  departTime,
			Origin:      origin,
			ArriveTime:  arriveTime,
			Destination: destination,
			VehicleName: vehicleName,
			SiteName:    siteName,
		}

		if direction == "outbound" {
			report.Outbound = append(report.Outbound, item)
		} else {
			report.Inbound = append(report.Inbound, item)
		}
	}

	return report, nil
}

// GenerateHsinchuScheduleExcel 產生符合規格書 §8.2 的新竹接送時刻表 Excel 檔案。
func (s *ReportService) GenerateHsinchuScheduleExcel(ctx context.Context, siteID *uuid.UUID, vehicleID *uuid.UUID) ([]byte, error) {
	report, err := s.GetHsinchuSchedule(ctx, siteID, vehicleID)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "新竹接送時刻表"
	defaultSheet := f.GetSheetName(0)
	f.SetSheetName(defaultSheet, sheetName)

	// 樣式定義
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "#1D5B79"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	sectionHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1D5B79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	runHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#1D5B79"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#EAF2F8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// 第 1 列：主標題
	f.SetCellValue(sheetName, "A1", "長照交通接送 新竹區搭車順序時刻表")
	f.MergeCell(sheetName, "A1", "H1")
	f.SetCellStyle(sheetName, "A1", "H1", titleStyle)

	currentRow := 3

	writeSection := func(directionTitle string, items []HsinchuScheduleItem) {
		if len(items) == 0 {
			return
		}

		// 標題列：去程/回程 | 趟次 | 姓名 | (備註) | 出發時間 | 出發地 | 抵達時間 | 目的地
		headers := []string{directionTitle, "趟次", "姓名", "備註", "出發時間", "出發地", "抵達時間", "目的地"}
		for colIdx, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, currentRow)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, sectionHeaderStyle)
		}
		currentRow++

		currentRun := int16(-1)
		for _, item := range items {
			var runText string
			if item.RunNo != currentRun {
				currentRun = item.RunNo
				runText = fmt.Sprintf("第%d趟", currentRun)
			} else {
				runText = "" // 僅在該趟第一列出現，其餘列留空（規格書 §8.2）
			}

			noteVal := ""
			if item.Note != nil {
				noteVal = *item.Note
			}
			arriveVal := ""
			if item.ArriveTime != nil {
				arriveVal = *item.ArriveTime
			}

			f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), directionTitle)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", currentRow), runText)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", currentRow), item.CaseName)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", currentRow), noteVal)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", currentRow), item.DepartTime)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", currentRow), item.Origin)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", currentRow), arriveVal)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", currentRow), item.Destination)

			if runText != "" {
				cell, _ := excelize.CoordinatesToCellName(2, currentRow)
				f.SetCellStyle(sheetName, cell, cell, runHeaderStyle)
			}
			currentRow++
		}
		currentRow += 2 // 區塊間隔
	}

	// 依序寫入去程與回程區塊
	writeSection("去程", report.Outbound)
	writeSection("回程", report.Inbound)

	// 設定欄寬
	f.SetColWidth(sheetName, "A", "B", 12)
	f.SetColWidth(sheetName, "C", "C", 16)
	f.SetColWidth(sheetName, "D", "D", 20)
	f.SetColWidth(sheetName, "E", "E", 14)
	f.SetColWidth(sheetName, "F", "F", 30)
	f.SetColWidth(sheetName, "G", "G", 14)
	f.SetColWidth(sheetName, "H", "H", 30)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write hsinchu schedule excel: %w", err)
	}

	return buf.Bytes(), nil
}
