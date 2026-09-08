package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"ltc-system/apps/api/internal/domain/namenorm"
)

// ParseCaregivers 僅支援解析 .xlsx 檔案，對齊「類型／單位／姓名／聯絡方式／備註」欄位格式。
func (s *CaregiverService) ParseCaregivers(ctx context.Context, r io.Reader, fileName string) (*CaregiverImportPreviewResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// 檢查是否為 Excel ZIP 格式 (Magic Number: PK\x03\x04)
	isExcel := len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04
	if !isExcel || !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
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
	CaregiverTypeSpecialist:  "照專",
}

// caregiverTypeFromLabel 依中文標籤比對類型代碼，找不到對應標籤回傳空字串，
// 由呼叫端以「未填寫」處理並列入待維護。
// 同時相容「照專」與舊稱「專護」。
func caregiverTypeFromLabel(label string) (code string, ok bool) {
	if label == "專護" {
		return CaregiverTypeSpecialist, true
	}
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

	for tableIdx, rows := range tables {
		sheetName := "Sheet"
		if tableIdx < len(sheetNames) && sheetNames[tableIdx] != "" {
			sheetName = sheetNames[tableIdx]
		}
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
			rowID := fmt.Sprintf("%s:%d", sheetName, actualRowIndex)
			rawValues := map[string]string{"單位": siteName, "姓名": name, "類型": typeLabel, "聯絡方式": contact, "備註": notes}

			// 姓名與類型缺漏不再擋列：以空白建立並列入待維護，讓使用者在待維護頁籤補齊，
			// 避免整列連同其他已填欄位一起被丟棄。類型比對不到固定選項時同樣存成空字串。
			typeCode, _ := caregiverTypeFromLabel(typeLabel)

			rowRes := CaregiverImportRowResult{RowID: rowID, RowIndex: actualRowIndex, SiteName: siteName, Name: name, Type: typeCode, Contact: contact, Notes: notes, RawValues: rawValues}

			if name == "" {
				rowRes.WarningMessage = appendCaregiverMessage(rowRes.WarningMessage, "姓名未填寫，將以空白建立並列入待維護")
			}
			if typeCode == "" {
				rowRes.WarningMessage = appendCaregiverMessage(rowRes.WarningMessage, "類型未填寫或不是「個管」／「照專」，將以空白建立並列入待維護")
			}
			// 單位比對到就自動關聯，比對不到只留白，不寫入原始名稱也不列入待維護；
			// 查詢本身失敗仍要中止，避免把「查詢故障」誤判成「查無單位」。
			if siteName != "" {
				if site, err := s.sites.GetByName(ctx, siteName); err == nil && site != nil {
					rowRes.SiteID = &site.ID
				} else if !errors.Is(err, ErrCaregiverSiteNotFound) && err != nil {
					return nil, fmt.Errorf("查詢單位「%s」失敗：%w", siteName, err)
				}
			}
			// 姓名為空時不查重：ILIKE '%%' 會撈回任意資料列，且正規化後的空字串會與既有
			// 空姓名資料互相命中，導致每一列都被誤判為重複而預設不匯入。
			if name != "" {
				// 重複人員不擋匯入，僅提示；使用者需於預覽勾選才會在正式匯入時寫入。
				dup, err := s.findDuplicateCaregiver(ctx, name)
				if err != nil {
					return nil, err
				}
				if dup != nil {
					rowRes.IsDuplicate = true
					rowRes.DuplicateCaregiverID = &dup.ID
					rowRes.DuplicateCaregiverName = dup.Name
					rowRes.WarningMessage = appendCaregiverMessage(rowRes.WarningMessage, fmt.Sprintf("疑似重複照護人員（既有資料「%s」），預設略過，需勾選才會匯入", dup.Name))
				}
			}

			validRows++
			results = append(results, rowRes)
			previewRow := map[string]interface{}{
				"rowId": rowID, "rowIndex": actualRowIndex, "siteName": siteName, "name": name, "type": typeLabel, "contact": contact, "notes": notes,
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

// findDuplicateCaregiver 以正規化姓名比對既有照護人員；資料庫查詢失敗時中止預覽，
// 避免把「查詢故障」誤判成「沒有重複」而放行匯入。
func (s *CaregiverService) findDuplicateCaregiver(ctx context.Context, name string) (*CaregiverDuplicateRef, error) {
	matches, _, err := s.store.List(ctx, name, "", false, false, 1, 5)
	if err != nil {
		return nil, fmt.Errorf("查詢照護人員重複資料失敗：%w", err)
	}
	normalized := namenorm.Normalize(name)
	for _, c := range matches {
		if namenorm.Normalize(c.Name) == normalized {
			return &CaregiverDuplicateRef{ID: c.ID, Name: c.Name}, nil
		}
	}
	return nil, nil
}

func appendCaregiverMessage(existing, next string) string {
	if existing == "" {
		return next
	}
	return existing + "；" + next
}

// CommitCaregivers 將解析出的照護人員資料正式寫入資料庫。姓名或類型缺漏的列同樣會寫入，
// 以空白值列入待維護供人工補齊；每一列各自獨立寫入，某列失敗只記為略過列，不影響其餘列。
// includeDuplicateRows 是使用者於預覽階段勾選「仍要匯入」的列號集合；標記為重複的列若未在
// 此集合中，直接記為略過。
func (s *CaregiverService) CommitCaregivers(ctx context.Context, preview *CaregiverImportPreviewResult, includeDuplicateRows map[string]bool, actors ...ActorContext) (*CaregiverImportCommitResult, error) {
	if preview == nil {
		return &CaregiverImportCommitResult{}, nil
	}

	result := &CaregiverImportCommitResult{}
	for _, errItem := range preview.Errors {
		result.SkippedRows = append(result.SkippedRows, CaregiverImportSkippedRow{
			RowID: errItem.RowID, RowIndex: errItem.RowIndex, Reasons: []string{errItem.Message},
		})
	}

	for _, row := range preview.Rows {
		if row.IsDuplicate && !includeDuplicateRows[caregiverRowKey(row.RowID, row.RowIndex)] {
			result.SkippedRows = append(result.SkippedRows, CaregiverImportSkippedRow{
				RowID: row.RowID, RowIndex: row.RowIndex, Name: row.Name, Reasons: []string{"偵測為重複人員，未勾選匯入"}, RawValues: row.RawValues,
			})
			continue
		}

		// 單位比對不到時保持空白，不保留原始名稱：單位已不是待維護的判定條件。
		c := Caregiver{Name: row.Name, Type: row.Type, Contact: row.Contact, Notes: row.Notes, SiteID: row.SiteID, Status: "active"}

		if err := s.store.Create(ctx, &c); err != nil {
			slog.Error("caregiver import row failed", "row_index", row.RowIndex, "error", err)
			result.SkippedRows = append(result.SkippedRows, CaregiverImportSkippedRow{
				RowID: row.RowID, RowIndex: row.RowIndex, Name: row.Name, Reasons: []string{"資料列匯入失敗，請檢查資料或稍後重試"}, RawValues: row.RawValues,
			})
			continue
		}

		result.ImportedCount++
		s.writeAudit(ctx, "import", c.ID, actorOrEmpty(actors), nil, c.AuditSnapshot())
		// 逐一依實際欄位狀態產生警告，而非拆解合併過的訊息字串，避免單列多項缺漏時遺漏分類。
		if row.Name == "" {
			result.Warnings = append(result.Warnings, CaregiverImportWarningItem{
				RowIndex: row.RowIndex, Name: row.Name, Field: "name", Message: "姓名未填寫，已以空白建立並列入待維護",
			})
		}
		if row.Type == "" {
			result.Warnings = append(result.Warnings, CaregiverImportWarningItem{
				RowIndex: row.RowIndex, Name: row.Name, Field: "type", Message: "類型未填寫或不是「個管」／「照專」，已以空白建立並列入待維護",
			})
		}
	}

	return result, nil
}

func caregiverRowKey(rowID string, rowIndex int) string {
	if rowID != "" {
		return rowID
	}
	return fmt.Sprintf("legacy:%d", rowIndex)
}

// CaregiverImportTemplateExcel 產生批次匯入標準 Excel 範本位元組。
func (s *CaregiverService) CaregiverImportTemplateExcel() ([]byte, error) {
	if s.renderer == nil {
		return nil, errors.New("caregiver import: template renderer not configured")
	}
	return s.renderer.RenderCaregiverImportTemplate()
}
