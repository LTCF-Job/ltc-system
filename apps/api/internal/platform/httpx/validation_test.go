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
