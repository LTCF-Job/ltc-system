package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// runHandler 以最小 gin engine 執行單一 handler，取得實際寫出的 response。
func runHandler(handler gin.HandlerFunc) *httptest.ResponseRecorder {
	engine := gin.New()
	engine.GET("/probe", handler)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/probe", nil))
	return w
}

// TestRespondSuccessNoContentHasEmptyBody 鎖住多個 DELETE handler 共用的
// RespondSuccess(c, 204, nil, nil) 寫法：HTTP 204 不得帶 body，gin 會在 Render 時丟棄
// APIResponse。若日後改寫成自行 Write 或換掉 gin 的 render 行為，會回傳含 body 的 204，
// 這在 HTTP 上是非法回應，部分 proxy 與前端會直接判為錯誤。
func TestRespondSuccessNoContentHasEmptyBody(t *testing.T) {
	w := runHandler(func(c *gin.Context) {
		RespondSuccess(c, http.StatusNoContent, nil, nil)
	})

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.Bytes(), "204 回應不得帶 body")
}

// TestRespondSuccessWritesEnvelope 確認一般成功回應仍維持 {data, meta} 外層結構，
// 這是前端 api client 解封裝的契約。
func TestRespondSuccessWritesEnvelope(t *testing.T) {
	w := runHandler(func(c *gin.Context) {
		RespondSuccess(c, http.StatusOK, gin.H{"id": "abc"}, PaginationMeta{Page: 1, PageSize: 20, Total: 1, TotalPages: 1})
	})

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Data map[string]any `json:"data"`
		Meta PaginationMeta `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "abc", got.Data["id"])
	assert.Equal(t, 1, got.Meta.Page)
	assert.Equal(t, int64(1), got.Meta.Total)
}

// TestRespondSuccessOmitsMetaWhenNil 確認未分頁的回應不會多出一個 meta 欄位。
func TestRespondSuccessOmitsMetaWhenNil(t *testing.T) {
	w := runHandler(func(c *gin.Context) {
		RespondSuccess(c, http.StatusOK, []string{}, nil)
	})

	require.Equal(t, http.StatusOK, w.Code)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	assert.Contains(t, raw, "data")
	assert.NotContains(t, raw, "meta")
}

// TestRespondErrorCodeHidesUnderlyingError 鎖住錯誤訊息單一事實來源：回傳給前端的 message
// 必須是 codeMessages 的固定文字，不得夾帶 Go／SQL 等底層錯誤內容。
func TestRespondErrorCodeHidesUnderlyingError(t *testing.T) {
	cases := []struct {
		name string
		code string
	}{
		{"驗證失敗", CodeValidationFailed},
		{"查無資料", CodeNotFound},
		{"系統錯誤", CodeInternalError},
		{"欄位對應失敗", CodeFormMappingFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := runHandler(func(c *gin.Context) {
				RespondErrorCode(c, http.StatusInternalServerError, tc.code, assert.AnError, nil)
			})

			var got ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			assert.Equal(t, tc.code, got.Error.Code)
			assert.Equal(t, codeMessages[tc.code], got.Error.Message)
			assert.NotContains(t, got.Error.Message, assert.AnError.Error())
		})
	}
}

// TestAllErrorCodesHaveMessage 防止新增錯誤碼時漏掉對應文字，導致前端顯示空白訊息。
// 逐一列舉常數會在新增錯誤碼時靜默失去防護，因此改為走 ErrorCodes() 這份對外清單。
func TestAllErrorCodesHaveMessage(t *testing.T) {
	codes := ErrorCodes()
	require.NotEmpty(t, codes)

	for _, code := range codes {
		assert.NotEmpty(t, MessageForCode(code), "錯誤碼 %s 缺少預設訊息", code)
	}
}

// TestErrorCodesCoversEveryDeclaredConstant 補上 ErrorCodes() 自己的漏網情形：
// 只要有錯誤碼常數沒被登記進 codeMessages，它就不會出現在 ErrorCodes() 裡，
// 上面那個測試也就掃不到它，最後在前端顯示成通用的「系統發生錯誤」。
func TestErrorCodesCoversEveryDeclaredConstant(t *testing.T) {
	declared := []string{
		CodeValidationFailed, CodeUnauthenticated, CodeForbidden, CodeNotFound,
		CodeAssignmentOverlap, CodeExportInProgress, CodePrecheckFailed, CodeNoExportData,
		CodeMappingRequired, CodeReportImportFailed, CodeFormMappingFailed,
		CodeInternalError, CodeServiceUnavailable, CodeResourceInUse,
		CodeUnsupportedFileType, CodeFileTooLarge, CodeFileUnreadable, CodeImportTemplateMismatch,
		CodeRouteNotFound,
	}
	registered := ErrorCodes()

	for _, code := range declared {
		assert.Contains(t, registered, code, "錯誤碼常數 %s 未登記於 codeMessages", code)
	}
}

// TestMessageForCodeFallsBackToInternalError 鎖住未知錯誤碼的降級行為。
func TestMessageForCodeFallsBackToInternalError(t *testing.T) {
	assert.Equal(t, codeMessages[CodeInternalError], MessageForCode("NOT_A_REAL_CODE"))
	assert.Equal(t, codeMessages[CodeInternalError], MessageForCode(""))
}

// TestRespondErrorFillsMessageFromCode 鎖住空訊息的降級行為：handler 傳空字串時
// 前端仍要拿到該錯誤碼的預設文字，而不是空白提示。
func TestRespondErrorFillsMessageFromCode(t *testing.T) {
	w := runHandler(func(c *gin.Context) {
		RespondError(c, http.StatusNotFound, CodeNotFound, "", nil)
	})

	var got ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, codeMessages[CodeNotFound], got.Error.Message)
}

// TestRespondErrorKeepsHandlerMessage 確認 handler 寫的具體訊息不會被錯誤碼預設文字蓋掉；
// 前端就是靠這個訊息顯示「哪裡錯」而不是通用提示。
func TestRespondErrorKeepsHandlerMessage(t *testing.T) {
	w := runHandler(func(c *gin.Context) {
		RespondError(c, http.StatusBadRequest, CodeValidationFailed, "月份格式錯誤，請使用 RRR-MM", nil)
	})

	var got ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "月份格式錯誤，請使用 RRR-MM", got.Error.Message)
}

// TestRespondErrorCarriesRequestID 確認錯誤回應帶回請求識別碼，讓使用者回報的畫面訊息
// 能對應到伺服器 log 的同一筆請求。
func TestRespondErrorCarriesRequestID(t *testing.T) {
	engine := gin.New()
	engine.Use(RequestIDMiddleware())
	engine.GET("/probe", func(c *gin.Context) {
		RespondError(c, http.StatusInternalServerError, CodeInternalError, "", nil)
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/probe", nil))

	var got ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.NotEmpty(t, got.Error.RequestID)
	assert.Equal(t, w.Header().Get(RequestIDHeader), got.Error.RequestID)
}
