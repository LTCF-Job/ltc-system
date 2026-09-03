package httpx

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxUploadBytes 是上傳檔案的請求主體上限。
// 20 MB 遠高於本系統實際匯入檔的規模（個案主檔數千列、司機匯報表單月一張工作表，
// 未壓縮亦不足 5 MB），同時把單一請求能佔用的記憶體壓在可控範圍。
const MaxUploadBytes = 20 << 20

// BindUploadFile 取出 multipart 表單欄位 field 的上傳檔案，並在超過 MaxUploadBytes
// 時直接回應 HTTP 413。回傳 false 代表已寫出錯誤回應，呼叫端應立即返回。
func BindUploadFile(c *gin.Context, field string) (*multipart.FileHeader, bool) {
	// 必須在解析 multipart 之前包住 body：只檢查 fileHeader.Size 時，Gin 早已把整份
	// 內容讀進記憶體或暫存檔，上限便形同虛設。
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes)

	fileHeader, err := c.FormFile(field)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondUploadTooLarge(c)
			return nil, false
		}
		RespondError(c, http.StatusBadRequest, CodeValidationFailed, "未提供上傳檔案", nil)
		return nil, false
	}
	if fileHeader.Size > MaxUploadBytes {
		respondUploadTooLarge(c)
		return nil, false
	}
	return fileHeader, true
}

func respondUploadTooLarge(c *gin.Context) {
	RespondError(c, http.StatusRequestEntityTooLarge, CodeValidationFailed, "上傳檔案超過 20 MB 上限，請分批匯入", nil)
}
