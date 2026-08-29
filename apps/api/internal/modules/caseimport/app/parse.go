package app

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"ltc-system/apps/api/internal/domain/crypto"
)

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
		tables, sheetNames, err := s.spreadsheet.ReadTables(data)
		if err != nil {
			return nil, err
		}
		return s.processRawTables(tables, sheetNames)
	}

	return s.parseCSVBytes(data)
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
	profileWorkbookFound := false
	for _, rows := range tables {
		for r := 0; r < min(3, len(rows)); r++ {
			rowText := strings.Join(rows[r], ",")
			if strings.Contains(rowText, "戶別") && strings.Contains(rowText, "居住地") {
				profileWorkbookFound = true
				break
			}
		}
	}

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
					if strings.Contains(cleanName, "接送車輛(去)") || strings.Contains(cleanName, "接送車輛（去）") {
						colMap["接送車輛(去)"] = c
					}
					if strings.Contains(cleanName, "接送車輛(回)") || strings.Contains(cleanName, "接送車輛（回）") {
						colMap["接送車輛(回)"] = c
					}
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
		isProfileWorkbook := false
		if _, ok := colMap["戶別"]; ok {
			_, isProfileWorkbook = colMap["居住地"]
		}
		if profileWorkbookFound && !isProfileWorkbook {
			continue
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
			rawValues := make(map[string]string)
			for label, index := range colMap {
				if index < len(row) {
					rawValues[label] = strings.TrimSpace(row[index])
				}
			}

			nationalID := getVal("身分證字號", "身分證")
			regionStr := getVal("申報地區", "地區", "區域")
			phone := getVal("聯絡電話", "電話")
			homeAddress := getVal("居住地", "住家地址", "地址")
			householdType := getVal("戶別")
			gender := getVal("性別")
			birthDate := parseProfileBirthDate(getVal("生日"))
			careContactRole := getVal("個管or照專", "個管／照專", "個管/照專")
			careContactName := getVal("聯絡人")
			registeredAddress := getVal("戶籍")
			outboundVehicle := getVal("接送車輛(去)")
			inboundVehicle := getVal("接送車輛(回)")
			claimStartDate := getVal("開始申報日", "幾號開始申報", "申報開始日")
			siteName := getVal("所屬據點", "據點", "據點名稱")
			distanceStr := getVal("單趟里程(公里)", "單趟里程", "里程", "里程數")
			unitPriceStr := getVal("申報單價(元)", "申報單價", "單價")
			durationStr := getVal("服務時長(分鐘)", "服務時長", "時長")
			serviceCatStr := getVal("服務類別")
			serviceUsageStr := getVal("服務使用類型")
			note := getVal("備註", "REMARK")

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
			if isProfileWorkbook {
				activeWeekdays = nil
				weekdayScheds = nil
				maxTripPattern = 0
				outboundTime = ""
				inboundTime = ""
			}

			rowRes := CaseImportRowResult{
				RowIndex:           actualRowIndex,
				SheetName:          sheetName,
				Name:               name,
				NationalID:         nationalID,
				Phone:              phone,
				HouseholdType:      householdType,
				Gender:             gender,
				BirthDate:          birthDate,
				CareContactRole:    careContactRole,
				CareContactName:    careContactName,
				RegisteredAddress:  registeredAddress,
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
				OutboundVehicle:    outboundVehicle,
				InboundVehicle:     inboundVehicle,
				TripPattern:        maxTripPattern,
				DistanceKM:         distanceKM,
				UnitPrice:          unitPrice,
				ServiceDurationMin: durationMin,
				Note:               note,
				IsProfileWorkbook:  isProfileWorkbook,
				IsDraft:            isProfileWorkbook || nationalID == "" || homeAddress == "",
				RawValues:          rawValues,
			}

			hasError := false
			hasWarning := false
			if isProfileWorkbook {
				requiredFields := []struct{ label, value string }{
					{"戶別", householdType}, {"身分證字號", nationalID}, {"性別", gender},
					{"據點", siteName}, {"接送車輛(去)", outboundVehicle}, {"接送車輛(回)", inboundVehicle},
					{"個管or照專", careContactRole}, {"聯絡人", careContactName}, {"戶籍", registeredAddress}, {"居住地", homeAddress},
				}
				for _, field := range requiredFields {
					if field.value == "" {
						message := field.label + "：空白"
						rowRes.ErrorMessage = strings.Trim(strings.TrimSpace(rowRes.ErrorMessage+"；"+message), "；")
						errorsList = append(errorsList, CaseImportErrorItem{RowIndex: actualRowIndex, CaseName: name, Field: field.label, Message: message})
						hasError = true
					}
				}
				if strings.TrimSpace(getVal("生日")) != "" && birthDate == "" {
					message := "生日：格式錯誤"
					rowRes.ErrorMessage = strings.Trim(strings.TrimSpace(rowRes.ErrorMessage+"；"+message), "；")
					errorsList = append(errorsList, CaseImportErrorItem{RowIndex: actualRowIndex, CaseName: name, Field: "生日", Message: message})
					hasError = true
				}
			}

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
				"rowIndex":          actualRowIndex,
				"name":              name,
				"nationalId":        crypto.Mask(nationalID),
				"region":            region,
				"claimStartDate":    claimStartDate,
				"siteName":          siteName,
				"weekdays":          wText,
				"departTime":        outboundTime,
				"returnTime":        inboundTime,
				"tripPattern":       maxTripPattern,
				"distanceKm":        rowRes.DistanceKM,
				"unitPrice":         unitPrice,
				"isDraft":           rowRes.IsDraft,
				"householdType":     householdType,
				"gender":            gender,
				"careContactRole":   careContactRole,
				"registeredAddress": registeredAddress,
				"__hasError":        hasError,
				"__hasWarning":      hasWarning,
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
