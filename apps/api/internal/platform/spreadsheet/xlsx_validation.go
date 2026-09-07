package spreadsheet

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

const (
	MaxXLSXZipEntries        = 1000
	MaxXLSXUncompressedBytes = 100 << 20
	MaxXLSXWorksheetXMLBytes = 30 << 20
	MaxXLSXCompressionRatio  = 1000
)

// ValidateXLSXZip 在交給 Excel parser 前檢查 ZIP 中央目錄，避免壓縮炸彈或
// 極端膨脹的 worksheet XML 先被完整解壓，繞過 HTTP 上傳大小限制。
func ValidateXLSXZip(data []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("xlsx zip 結構無效: %w", err)
	}
	if len(reader.File) > MaxXLSXZipEntries {
		return fmt.Errorf("xlsx ZIP 項目數超過上限 %d", MaxXLSXZipEntries)
	}

	var totalUncompressed uint64
	for _, file := range reader.File {
		size := file.UncompressedSize64
		if size > MaxXLSXUncompressedBytes || totalUncompressed > MaxXLSXUncompressedBytes-size {
			return fmt.Errorf("xlsx 解壓後大小超過上限 %d bytes", MaxXLSXUncompressedBytes)
		}
		totalUncompressed += size

		name := strings.ToLower(file.Name)
		if strings.HasPrefix(name, "xl/worksheets/") && size > MaxXLSXWorksheetXMLBytes {
			return fmt.Errorf("xlsx 工作表 XML 超過上限 %d bytes", MaxXLSXWorksheetXMLBytes)
		}

		compressed := file.CompressedSize64
		if compressed == 0 {
			if size > 0 {
				return fmt.Errorf("xlsx ZIP 項目壓縮大小無效: %s", file.Name)
			}
			continue
		}
		if size/compressed > MaxXLSXCompressionRatio {
			return fmt.Errorf("xlsx ZIP 壓縮倍率超過上限 %d: %s", MaxXLSXCompressionRatio, file.Name)
		}
	}
	return nil
}
