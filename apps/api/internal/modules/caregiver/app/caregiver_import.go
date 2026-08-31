package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"ltc-system/apps/api/internal/domain/namenorm"
)

// ParseCaregivers 僅支援解析 .xlsx／.xls 檔案，對齊「類型／單位／姓名／聯絡方式／備註」欄位格式。
func (s *CaregiverService) ParseCaregivers(ctx context.Context, r io.Reader, fileName string) (*CaregiverImportPreviewResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// 檢查是否為 Excel ZIP 格式 (Magic Number: PK\x03\x04)
	isExcel := len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04
	if !isExcel && !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") && !strings.HasSuffix(strings.ToLower(fileName), ".xls") {
		return nil, errors.New("僅支援 .xlsx 匯入格式")
	}
	if s.reader == nil {
		return nil, errors.New("caregiver import: spreadsheet reader not configured")
	}
	tables, sheetNames, err := s.reader.ReadTables(data)
	if err != nil {
		return nil, err
	}
	return s.processRawTables(ctx, tables, sheetNames)
}

// caregiverColumns 是表頭關鍵字對應的欄位名稱，依序嘗試比對第一列儲存格內容。
var caregiverColumns = map[string][]string{
	"site":    {"單位"},
	"name":    {"姓名"},
	"type":    {"類型"},
	"contact": {"聯絡方式"},
	"notes":   {"備註"},
}

// caregiverTypeLabels 是類型固定選項的中文標籤，供匯入比對與範本顯示使用。
var caregiverTypeLabels = map[string]string{
	CaregiverTypeCaseManager: "個管",
	CaregiverTypeSpecialist:  "專護",
}

// caregiverTypeFromLabel 依中文標籤比對類型代碼，找不到對應標籤視為未填寫或格式錯誤。
func caregiverTypeFromLabel(label string) (code string, ok bool) {
	for c, l := range caregiverTypeLabels {
		if l == label {
			return c, true
		}
	}
	return "", false
}

// findCaregiverHeader 在工作表第一列尋找標題列，回傳各欄位名稱對應的欄位索引。
func findCaregiverHeader(rows [][]string) (colMap map[string]int, ok bool) {
	if len(rows) == 0 {
		return nil, false
	}
	colMap = make(map[string]int)
	for c, cell := range rows[0] {
		cleanName := strings.TrimSpace(strings.ReplaceAll(cell, "*", ""))
		for field, keywords := range caregiverColumns {
			for _, kw := range keywords {
				if cleanName == kw {
					colMap[field] = c
				}
			}
		}
	}
	_, hasName := colMap["name"]
	return colMap, hasName
}

func (s *CaregiverService) processRawTables(ctx context.Context, tables [][][]string, sheetNames []string) (*CaregiverImportPreviewResult, error) {
	var results []CaregiverImportRowResult
	var errorsList []CaregiverImportErrorItem
	var previewRows []map[string]interface{}

	totalRows := 0
	validRows := 0

	for _, rows := range tables {
		colMap, ok := findCaregiverHeader(rows)
		if !ok {
			continue
		}

		getVal := func(row []string, field string) string {
			if idx, ok := colMap[field]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}

		for rIdx := 1; rIdx < len(rows); rIdx++ {
			row := rows[rIdx]
			if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
				continue
			}

			siteName := getVal(row, "site")
			name := getVal(row, "name")
			typeLabel := getVal(row, "type")
			contact := getVal(row, "contact")
			notes := getVal(row, "notes")
			if siteName == "" && name == "" && typeLabel == "" && contact == "" && notes == "" {
				continue
			}

			totalRows++
			actualRowIndex := rIdx + 1
			rawValues := map[string]string{"單位": siteName, "姓名": name, "類型": typeLabel, "聯絡方式": contact, "備註": notes}

			// 姓名與類型為必填欄位，缺漏或類型不是「個管」／「專護」即整列略過，不進入可匯入的預覽列。
			if name == "" {
				message := "姓名：未填寫，本列已略過"
				errorsList = append(errorsList, CaregiverImportErrorItem{RowIndex: actualRowIndex, Field: "姓名", Message: message})
				previewRows = append(previewRows, map[string]interface{}{
					"rowIndex": actualRowIndex, "siteName": siteName, "name": name, "type": typeLabel, "contact": contact, "notes": notes,
					"__hasError": true, "__hasWarning": false,
				})
				continue
			}
			typeCode, validType := caregiverTypeFromLabel(typeLabel)
			if !validType {
				message := "類型：未填寫或不是「個管」／「專護」，本列已略過"
				errorsList = append(errorsList, CaregiverImportErrorItem{RowIndex: actualRowIndex, Name: name, Field: "類型", Message: message})
				previewRows = append(previewRows, map[string]interface{}{
					"rowIndex": actualRowIndex, "siteName": siteName, "name": name, "type": typeLabel, "contact": contact, "notes": notes,
					"__hasError": true, "__hasWarning": false,
				})
				continue
			}

			rowRes := CaregiverImportRowResult{RowIndex: actualRowIndex, SiteName: siteName, Name: name, Type: typeCode, Contact: contact, Notes: notes, RawValues: rawValues}

			// 單位、聯絡方式、備註缺漏或比對不到都不擋匯入，僅提示待後續維護。
			if siteName != "" {
				if site, err := s.sites.GetByName(ctx, siteName); err == nil && site != nil {
					rowRes.SiteID = &site.ID
				} else {
					rowRes.WarningMessage = appendCaregiverMessage(rowRes.WarningMessage, fmt.Sprintf("單位「%s」未於據點管理中找到，已建立資料並保留原始名稱待人工關聯", siteName))
				}
			}
			if contact == "" {
				rowRes.WarningMessage = appendCaregiverMessage(rowRes.WarningMessage, "聯絡方式未填寫，已建立資料待後續補齊")
			}
			if notes == "" {
				rowRes.WarningMessage = appendCaregiverMessage(rowRes.WarningMessage, "備註未填寫，已建立資料待後續補齊")
			}
			// 重複人員不擋匯入，僅提示；使用者需於預覽勾選才會在正式匯入時寫入。
			if dup := s.findDuplicateCaregiver(ctx, name); dup != nil {
				rowRes.IsDuplicate = true
				rowRes.DuplicateCaregiverID = &dup.ID
				rowRes.DuplicateCaregiverName = dup.Name
				rowRes.WarningMessage = appendCaregiverMessage(rowRes.WarningMessage, fmt.Sprintf("疑似重複照護人員（既有資料「%s」），預設略過，需勾選才會匯入", dup.Name))
			}

			validRows++
			results = append(results, rowRes)
			previewRow := map[string]interface{}{
				"rowIndex": actualRowIndex, "siteName": siteName, "name": name, "type": typeLabel, "contact": contact, "notes": notes,
				"isDuplicate": rowRes.IsDuplicate, "__hasError": false, "__hasWarning": rowRes.WarningMessage != "",
			}
			if rowRes.IsDuplicate {
				previewRow["duplicateOf"] = map[string]string{"name": rowRes.DuplicateCaregiverName}
			}
			previewRows = append(previewRows, previewRow)
		}

		_ = sheetNames
	}

	warningRows := 0
	for _, row := range results {
		if row.WarningMessage != "" {
			warningRows++
		}
	}

	return &CaregiverImportPreviewResult{
		TotalRows:   totalRows,
		ValidRows:   validRows,
		ErrorRows:   totalRows - validRows,
		WarningRows: warningRows,
		PreviewRows: previewRows,
		Errors:      errorsList,
		Rows:        results,
	}, nil
}

// findDuplicateCaregiver 以正規化姓名比對既有照護人員；查詢失敗時視為無重複，不中斷整批解析。
func (s *CaregiverService) findDuplicateCaregiver(ctx context.Context, name string) *CaregiverDuplicateRef {
	matches, _, err := s.store.List(ctx, name, false, false, false, 1, 5)
	if err != nil {
		return nil
	}
	normalized := namenorm.Normalize(name)
	for _, c := range matches {
		if namenorm.Normalize(c.Name) == normalized {
			return &CaregiverDuplicateRef{ID: c.ID, Name: c.Name}
		}
	}
	return nil
}

func appendCaregiverMessage(existing, next string) string {
	if existing == "" {
		return next
	}
	return existing + "；" + next
}

// CommitCaregivers 將通過檢核的照護人員資料正式寫入資料庫。姓名缺漏的列已於
// dry-run 階段排除在 preview.Rows 之外；每一列各自獨立寫入，某列失敗只記為
// 略過列，不影響其餘列的匯入。includeDuplicateRows 是使用者於預覽階段勾選
// 「仍要匯入」的列號集合；標記為重複的列若未在此集合中，直接記為略過。
func (s *CaregiverService) CommitCaregivers(ctx context.Context, preview *CaregiverImportPreviewResult, includeDuplicateRows map[int]bool) (*CaregiverImportCommitResult, error) {
	if preview == nil {
		return &CaregiverImportCommitResult{}, nil
	}

	result := &CaregiverImportCommitResult{}
	for _, errItem := range preview.Errors {
		result.SkippedRows = append(result.SkippedRows, CaregiverImportSkippedRow{
			RowIndex: errItem.RowIndex, Reasons: []string{errItem.Message},
		})
	}

	for _, row := range preview.Rows {
		if row.IsDuplicate && !includeDuplicateRows[row.RowIndex] {
			result.SkippedRows = append(result.SkippedRows, CaregiverImportSkippedRow{
				RowIndex: row.RowIndex, Name: row.Name, Reasons: []string{"偵測為重複人員，未勾選匯入"}, RawValues: row.RawValues,
			})
			continue
		}

		c := Caregiver{Name: row.Name, Type: row.Type, Contact: row.Contact, Notes: row.Notes, SiteID: row.SiteID}
		if row.SiteID == nil {
			c.SiteNameRaw = row.SiteName
		}

		if err := s.store.Create(ctx, &c); err != nil {
			result.SkippedRows = append(result.SkippedRows, CaregiverImportSkippedRow{
				RowIndex: row.RowIndex, Name: row.Name, Reasons: []string{err.Error()}, RawValues: row.RawValues,
			})
			continue
		}

		result.ImportedCount++
		// 逐一依實際欄位狀態產生警告，而非拆解合併過的訊息字串，避免單列多項缺漏時遺漏分類。
		if row.SiteID == nil && row.SiteName != "" {
			result.Warnings = append(result.Warnings, CaregiverImportWarningItem{
				RowIndex: row.RowIndex, Name: row.Name, Field: "site",
				Message: fmt.Sprintf("單位「%s」未於據點管理中找到，已建立資料並保留原始名稱待人工關聯", row.SiteName),
			})
		}
		if row.Contact == "" {
			result.Warnings = append(result.Warnings, CaregiverImportWarningItem{
				RowIndex: row.RowIndex, Name: row.Name, Field: "contact", Message: "聯絡方式未填寫，已建立資料待後續補齊",
			})
		}
		if row.Notes == "" {
			result.Warnings = append(result.Warnings, CaregiverImportWarningItem{
				RowIndex: row.RowIndex, Name: row.Name, Field: "notes", Message: "備註未填寫，已建立資料待後續補齊",
			})
		}
	}

	return result, nil
}

// CaregiverImportTemplateExcel 產生批次匯入標準 Excel 範本位元組。
func (s *CaregiverService) CaregiverImportTemplateExcel() ([]byte, error) {
	if s.renderer == nil {
		return nil, errors.New("caregiver import: template renderer not configured")
	}
	return s.renderer.RenderCaregiverImportTemplate()
}
