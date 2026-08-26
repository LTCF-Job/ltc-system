package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDownloadTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &CaseHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/cases/template", nil)

	h.DownloadTemplate(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("expected Content-Type to contain text/csv, got %s", contentType)
	}

	contentDisposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "個案批次匯入範本.csv") {
		t.Errorf("expected Content-Disposition to contain filename, got %s", contentDisposition)
	}

	body := w.Body.String()
	if !strings.HasPrefix(body, "\uFEFF個案姓名*") {
		t.Errorf("expected body to start with UTF-8 BOM and header, got %q", body[:min(len(body), 30)])
	}
}
