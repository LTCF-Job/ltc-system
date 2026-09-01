package infra

import (
	"archive/zip"
	"bytes"
	"fmt"

	"ltc-system/apps/api/internal/modules/reporting/app"
)

// ZipArchiver 以標準函式庫將多份工作簿打包成單一壓縮檔。
type ZipArchiver struct{}

// NewZipArchiver 建立 ZipArchiver 實例。
func NewZipArchiver() ZipArchiver { return ZipArchiver{} }

// BuildZip 依序寫入各檔案並回傳壓縮檔位元組。
func (ZipArchiver) BuildZip(entries []app.ZipEntry) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)

	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.Name, Method: zip.Deflate}
		// 個案檔名含中文，標記 UTF-8 旗標讓解壓縮工具不以本機字碼頁解讀
		header.Flags |= 0x800

		file, err := writer.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("create zip entry %q: %w", entry.Name, err)
		}
		if _, err := file.Write(entry.Content); err != nil {
			return nil, fmt.Errorf("write zip entry %q: %w", entry.Name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close zip writer: %w", err)
	}
	return buf.Bytes(), nil
}
