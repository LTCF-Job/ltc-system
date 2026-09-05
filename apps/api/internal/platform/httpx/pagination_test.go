package httpx

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePaginationRejectsInvalidExplicitValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, query := range []string{"?page=0", "?page=-1", "?pageSize=0", "?pageSize=101", "?pageSize=oops"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/items"+query, nil)
		if _, _, err := ParsePagination(c); err == nil {
			t.Errorf("query %s: expected error", query)
		}
	}
}

func TestParsePaginationUsesDefaults(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/items", nil)
	page, pageSize, err := ParsePagination(c)
	if err != nil || page != 1 || pageSize != 20 {
		t.Fatalf("got (%d, %d, %v), want (1, 20, nil)", page, pageSize, err)
	}
}
