package httpx

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueryBool 鎖住待維護旗標的解析行為：strconv 認得的寫法一律成立，其餘一律當作未表態。
// 兩邊 handler 共用這個函式，這裡是唯一需要驗證解析規則的地方。
func TestQueryBool(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		query string
		want  bool
	}{
		{"?flag=true", true},
		{"?flag=TRUE", true},
		{"?flag=True", true},
		{"?flag=1", true},
		{"?flag=t", true},
		{"?flag=false", false},
		{"?flag=0", false},
		{"?flag=", false},
		{"", false},
		{"?flag=yes", false},
		{"?flag=on", false},
		{"?other=true", false},
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/probe"+tc.query, nil)

			assert.Equal(t, tc.want, QueryBool(c, "flag"))
		})
	}
}

type sampleRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address" binding:"required"`
	Age     int    `json:"age" binding:"min=18"`
}

func TestExtractValidationDetails_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var details []ErrorDetail

	r.POST("/test", func(c *gin.Context) {
		var req sampleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			details = ExtractValidationDetails(err)
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"age": 10}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NotEmpty(t, details)

	// 應抓出 name, address 必填與 age 的 min 限制
	fields := make(map[string]string)
	for _, d := range details {
		fields[d.Field] = d.Reason
	}

	assert.Contains(t, fields, "name")
	assert.Contains(t, fields["name"], "必填")
	assert.Contains(t, fields, "address")
	assert.Contains(t, fields["address"], "必填")
	assert.Contains(t, fields, "age")
	assert.Contains(t, fields["age"], "不得小於 18")
}

func TestExtractValidationDetails_TypeError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var details []ErrorDetail

	r.POST("/test", func(c *gin.Context) {
		var req sampleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			details = ExtractValidationDetails(err)
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name": "test", "address": "test", "age": "not-a-number"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Len(t, details, 1)
	assert.Equal(t, "age", details[0].Field)
	assert.Contains(t, details[0].Reason, "資料型態錯誤")
}

func TestExtractValidationDetails_EOF(t *testing.T) {
	details := ExtractValidationDetails(io.EOF)
	require.Len(t, details, 1)
	assert.Contains(t, details[0].Reason, "不能為空")
}

func TestExtractValidationDetails_SyntaxError(t *testing.T) {
	var target struct{}
	err := json.Unmarshal([]byte(`{invalid json`), &target)
	details := ExtractValidationDetails(err)
	require.Len(t, details, 1)
	assert.Contains(t, details[0].Reason, "JSON 格式錯誤")
}
