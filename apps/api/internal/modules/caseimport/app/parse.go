package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"ltc-system/apps/api/internal/domain/crypto"
)

// 匯入前置失敗的 sentinel：transport layer 靠它們把「檔案類型不對」「檔案讀不出來」
// 「欄位跟範本不合」對應到各自的錯誤碼，使用者才知道該換檔案還是改欄位。
var (
	ErrUnsupportedFileType = errors.New("caseimport: unsupported file type")
	ErrFileUnreadable      = errors.New("caseimport: file unreadable")
	ErrTemplateMismatch    = errors.New("caseimport: template header mismatch")
)

// ParseCasesFromExcel 保留相容介面，實作通用串流解析。
func (s *ImportService) ParseCasesFromExcel(ctx context.Context, r io.Reader) (*CaseImportPreviewResult, error) {
	return s.ParseCases(ctx, r, "upload.xlsx")
}

// ParseCases 僅支援解析 .xlsx 檔案，對齊「進系統個案個資」欄位格式。
func (s *ImportService) ParseCases(ctx context.Context, r io.Reader, fileName string) (*CaseImportPreviewResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFileUnreadable, err)
	}

	// 檢查是否為 Excel ZIP 格式 (Magic Number: PK\x03\x04)
	isExcel := len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04
	if !isExcel || !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		return nil, ErrUnsupportedFileType
	}
	fileHash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))

	tables, sheetNames, err := s.spreadsheet.ReadTables(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFileUnreadable, err)
	}
	preview, err := s.processRawTables(ctx, tables, sheetNames)
	if err != nil {
		return nil, err
	}
	preview.FileHash = fileHash
	return preview, nil
}

// findHeader 在工作表前 3 列尋找標題列，解析出個案姓名欄的位置；其餘欄位（含「照護人員」）
// 名稱皆唯一，直接落在 colMap 即可，不需要再靠位置消歧。
func findHeader(rows [][]string) (headerRowIdx int, colMap map[string]int, caseNameIdx int) {
	colMap = make(map[string]int)
	caseNameIdx = -1

	for r := 0; r < min(3, len(rows)); r++ {
		rowText := strings.Join(rows[r], ",")
		if !strings.Contains(rowText, "姓名") {
			continue
		}

		for c, colName := range rows[r] {
			cleanName := strings.TrimSpace(strings.ReplaceAll(colName, "*", ""))
			cleanName = strings.Split(cleanName, "(")[0]
			cleanName = strings.Split(cleanName, "（")[0]
			cleanName = strings.TrimSpace(cleanName)
			if cleanName == "" {
				continue
			}
			if cleanName == "姓名" {
				if caseNameIdx == -1 {
					caseNameIdx = c
				}
				continue
			}
			if _, exists := colMap[cleanName]; !exists {
				colMap[cleanName] = c
			}
		}

		if caseNameIdx >= 0 {
			return r, colMap, caseNameIdx
		}
	}

	return 0, colMap, -1
}

func (s *ImportService) processRawTables(ctx context.Context, tables [][][]string, sheetNames []string) (*CaseImportPreviewResult, error) {
	var results []CaseImportRowResult
	var errorsList []CaseImportErrorItem
	var warningsList []CaseImportWarningItem
	var previewRows []map[string]interface{}

	totalRows := 0
	validRows := 0
	errorRows := 0
	warningRows := 0

	// 沒有任何工作表找得到「姓名」欄時，整份檔案會解析出零列。靜默回傳空結果會讓畫面
	// 顯示「總筆數 0」而說不出原因，因此記下曾檢查過哪些工作表，最後改回傳範本不符。
	var inspectedSheets []string
	headerFound := false

	for tableIdx, rows := range tables {
		sheetName := sheetNames[tableIdx]
		if len(rows) < 1 {
			continue
		}

		headerRowIdx, colMap, caseNameIdx := findHeader(rows)
		if caseNameIdx < 0 {
			inspectedSheets = append(inspectedSheets, sheetName)
			continue
		}
		headerFound = true

		for rIdx := headerRowIdx + 1; rIdx < len(rows); rIdx++ {
			row := rows[rIdx]
			if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
				continue
			}

			getVal := func(key string) string {
				if idx, ok := colMap[key]; ok && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}
			getIdxVal := func(idx int) string {
				if idx >= 0 && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}

			name := getIdxVal(caseNameIdx)
			// 個案範本的示範列已改為不帶前綴的虛構資料，這裡只剩下相容既有檔案的用途：
			// 使用者手上仍可能留著舊版範本，或自行以「例：」標註不要匯入的列。
			if strings.HasPrefix(name, "例:") || strings.HasPrefix(name, "例：") {
				continue
			}
			if name == "" {
				// 整列皆空是表尾補白；有其他欄位填了值代表使用者漏填姓名，靜默略過會讓整列無聲消失。
				if isBlankRow(row) {
					continue
				}
				totalRows++
				errorRows++
				blankNameRowIndex := rIdx + 1
				blankNameRowID := fmt.Sprintf("%s:%d", sheetName, blankNameRowIndex)
				errorsList = append(errorsList, CaseImportErrorItem{
					RowID:    blankNameRowID,
					RowIndex: blankNameRowIndex,
					Field:    "姓名",
					Message:  "姓名未填寫，此列不會匯入",
				})
				results = append(results, CaseImportRowResult{
					RowID:        blankNameRowID,
					RowIndex:     blankNameRowIndex,
					SheetName:    sheetName,
					ErrorMessage: "姓名未填寫，此列不會匯入",
				})
				previewRows = append(previewRows, map[string]interface{}{
					"rowId":      blankNameRowID,
					"rowIndex":   blankNameRowIndex,
					"name":       "",
					"__hasError": true,
				})
				continue
			}

			totalRows++
			actualRowIndex := rIdx + 1
			rowID := fmt.Sprintf("%s:%d", sheetName, actualRowIndex)
			rawValues := make(map[string]string)
			for label, index := range colMap {
				if index < len(row) {
					rawValues[label] = strings.TrimSpace(row[index])
				}
			}

			nationalID := getVal("身分證字號")
			householdType := getVal("戶別")
			gender := getVal("性別")
			birthDate := parseProfileBirthDate(getVal("生日"))
			siteName := getVal("據點")
			careContactRole := getVal("個管or照專")
			careContactName := getVal("照護人員")
			registeredAddress := getVal("戶籍")
			homeAddress := getVal("居住地")
			remarks := getVal("備註")
			if remarks == "" {
				remarks = getVal("REMARK")
			}

			rowRes := CaseImportRowResult{
				RowID:             rowID,
				RowIndex:          actualRowIndex,
				SheetName:         sheetName,
				Name:              name,
				NationalID:        nationalID,
				HouseholdType:     householdType,
				Gender:            gender,
				BirthDate:         birthDate,
				CareContactRole:   careContactRole,
				CareContactName:   careContactName,
				RegisteredAddress: registeredAddress,
				HomeAddress:       homeAddress,
				SiteName:          siteName,
				Remarks:           remarks,
				RawValues:         rawValues,
			}

			hasError := false
			hasWarning := false

			// 生日格式錯誤不擋列：個案照常建立，birth_date 留空、原始字串存 birth_date_raw，
			// 由使用者於待維護頁就地補正（比照據點/車輛比對不到主檔的既有待維護模式）。
			if strings.TrimSpace(getVal("生日")) != "" && birthDate == "" {
				rowRes.BirthDateInvalid = true
				rowRes.BirthDateRaw = getVal("生日")
				message := "生日：格式錯誤，將建立個案並標記待補正"
				rowRes.WarningMessage = appendMessage(rowRes.WarningMessage, message)
				warningsList = append(warningsList, CaseImportWarningItem{RowID: rowID, RowIndex: actualRowIndex, CaseName: name, Field: "生日", Message: message})
				hasWarning = true
			}

			normalizedNationalID := strings.ToUpper(strings.TrimSpace(nationalID))
			// 身分證字號格式錯誤同樣不擋列，也不保留原始錯誤字串（未通過格式驗證的字串
			// 不套用加密管線）；個案標記待補正，使用者需於待維護頁重新完整輸入。
			if normalizedNationalID != "" && !crypto.ValidateNationalID(normalizedNationalID) {
				rowRes.NationalIDInvalid = true
				message := "身分證字號：格式錯誤，將建立個案並標記待補正（需於待維護頁重新輸入）"
				rowRes.WarningMessage = appendMessage(rowRes.WarningMessage, message)
				warningsList = append(warningsList, CaseImportWarningItem{RowID: rowID, RowIndex: actualRowIndex, CaseName: name, Field: "身分證字號", Message: message})
				hasWarning = true
			}

			// 重複個案不擋匯入；正式匯入時會建立為待裁決暫存列，不會直接建立個案。
			// 格式錯誤的身分證字號不會被寫入，拿它算 HMAC 必定比不到任何個案，還會蓋掉
			// 姓名比對；這一列實際上等同「沒有身分證字號」，比對鍵也要一致
			duplicateLookupNationalID := normalizedNationalID
			if rowRes.NationalIDInvalid {
				duplicateLookupNationalID = ""
			}
			if !hasError && s.duplicates != nil {
				dup, err := s.duplicates.FindDuplicate(ctx, duplicateLookupNationalID, name)
				if err != nil {
					message := "重複個案查詢失敗，請稍後重試"
					rowRes.ErrorMessage = appendMessage(rowRes.ErrorMessage, message)
					errorsList = append(errorsList, CaseImportErrorItem{RowID: rowID, RowIndex: actualRowIndex, CaseName: name, Field: "重複個案", Message: message})
					hasError = true
				} else if dup != nil {
					rowRes.IsDuplicate = true
					rowRes.DuplicateCaseName = dup.CaseName
					rowRes.DuplicateCaseID = &dup.CaseID
					message := fmt.Sprintf("疑似重複個案（既有個案姓名 %s），正式匯入時將建立為待裁決項目，不會直接建立個案", dup.CaseName)
					rowRes.WarningMessage = appendMessage(rowRes.WarningMessage, message)
					warningsList = append(warningsList, CaseImportWarningItem{RowID: rowID, RowIndex: actualRowIndex, CaseName: name, Field: "重複個案", Message: message})
					hasWarning = true
				}
			}

			// 照護人員比對不到主檔不擋列；沿用 commit 階段同一套 resolveCaregiver，讓預覽
			// 就能提示使用者「正式匯入後會落入待維護」，不用等到正式匯入才發現。
			if !hasError {
				caregiverID, _, caregiverWarning, err := s.resolveCaregiver(ctx, careContactName, careContactRole)
				if err != nil {
					message := "照護人員查詢失敗，請稍後重試"
					rowRes.ErrorMessage = appendMessage(rowRes.ErrorMessage, message)
					errorsList = append(errorsList, CaseImportErrorItem{RowID: rowID, RowIndex: actualRowIndex, CaseName: name, Field: "照護人員", Message: message})
					hasError = true
				} else if caregiverID == nil && caregiverWarning != "" {
					rowRes.CaregiverUnmatched = true
					rowRes.WarningMessage = appendMessage(rowRes.WarningMessage, caregiverWarning)
					warningsList = append(warningsList, CaseImportWarningItem{RowID: rowID, RowIndex: actualRowIndex, CaseName: name, Field: "照護人員", Message: caregiverWarning})
					hasWarning = true
				}
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

			previewRow := map[string]interface{}{
				"rowId":              rowID,
				"rowIndex":           actualRowIndex,
				"name":               name,
				"nationalId":         crypto.Mask(nationalID),
				"householdType":      householdType,
				"gender":             gender,
				"birthDate":          birthDate,
				"siteName":           siteName,
				"careContactRole":    careContactRole,
				"careContactName":    careContactName,
				"registeredAddress":  registeredAddress,
				"homeAddress":        homeAddress,
				"remarks":            remarks,
				"isDuplicate":        rowRes.IsDuplicate,
				"birthDateInvalid":   rowRes.BirthDateInvalid,
				"nationalIdInvalid":  rowRes.NationalIDInvalid,
				"caregiverUnmatched": rowRes.CaregiverUnmatched,
				"__hasError":         hasError,
				"__hasWarning":       hasWarning,
			}
			if rowRes.IsDuplicate {
				previewRow["duplicateOf"] = map[string]string{
					"name": rowRes.DuplicateCaseName,
				}
			}
			previewRows = append(previewRows, previewRow)
		}
	}

	if !headerFound {
		return nil, fmt.Errorf("%w: 工作表 %s 找不到「姓名」欄", ErrTemplateMismatch, strings.Join(inspectedSheets, "、"))
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

func isBlankRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func appendMessage(existing, next string) string {
	if existing == "" {
		return next
	}
	return existing + "；" + next
}

func parseProfileBirthDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.Format("2006-01-02")
	}
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '-' || r == '.' })
	if len(parts) != 3 {
		return ""
	}
	year, yearErr := strconv.Atoi(parts[0])
	month, monthErr := strconv.Atoi(parts[1])
	day, dayErr := strconv.Atoi(parts[2])
	if yearErr != nil || monthErr != nil || dayErr != nil {
		return ""
	}
	if year < 1911 {
		year += 1911
	}
	parsed := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if parsed.Year() != year || int(parsed.Month()) != month || parsed.Day() != day {
		return ""
	}
	return parsed.Format("2006-01-02")
}
