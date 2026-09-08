package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
	"ltc-system/apps/api/internal/platform/config"
)

// TestCaseHandler_List_PendingFilterDefaults 鎖住待維護個案的預設可見性：呼叫端沒有明確
// 表態時一律排除，避免新增的清單頁忘記帶參數就把待維護資料洩漏出去。
func TestCaseHandler_List_PendingFilterDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name  string
		query string
		want  caseListFlags
	}{
		{"未帶參數預設排除待維護", "", caseListFlags{unresolvedLink: false, excludePending: true}},
		{"includePending=true 取全部", "?includePending=true", caseListFlags{unresolvedLink: false, excludePending: false}},
		{"unresolvedLink=true 只取待維護", "?unresolvedLink=true", caseListFlags{unresolvedLink: true, excludePending: false}},
		{"兩者同時帶時以待維護清單為準", "?unresolvedLink=true&includePending=true", caseListFlags{unresolvedLink: true, excludePending: false}},
		// 與 /caregivers 共用 httpx.QueryBool，1／TRUE 等 strconv 認得的寫法在兩邊都有效。
		{"includePending=1 等同 true", "?includePending=1", caseListFlags{unresolvedLink: false, excludePending: false}},
		{"unresolvedLink=TRUE 等同 true", "?unresolvedLink=TRUE", caseListFlags{unresolvedLink: true, excludePending: false}},
		{"無法解析的值當作未表態", "?includePending=yes", caseListFlags{unresolvedLink: false, excludePending: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeCaseStore{}
			h := newTestCaseHandler(store)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cases"+tc.query, nil)

			h.List(c)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tc.want, store.listFlags)
		})
	}
}

// fakeDuplicateStaging 只回應 handler 錯誤對應測試需要的兩個結果。
type fakeDuplicateStaging struct {
	candidate  *app.DuplicateCandidate
	deleteRows int64
}

func (f *fakeDuplicateStaging) Insert(context.Context, app.DuplicateCandidate) (uuid.UUID, bool, error) {
	return uuid.Nil, false, nil
}

func (f *fakeDuplicateStaging) ListPending(context.Context) ([]app.DuplicateCandidate, error) {
	return nil, nil
}

func (f *fakeDuplicateStaging) GetByID(context.Context, uuid.UUID) (*app.DuplicateCandidate, error) {
	return f.candidate, nil
}

func (f *fakeDuplicateStaging) Resolve(context.Context, uuid.UUID, string, uuid.UUID, *uuid.UUID) (int64, error) {
	return 0, errors.New("not used")
}

func (f *fakeDuplicateStaging) Delete(context.Context, uuid.UUID) (int64, error) {
	return f.deleteRows, nil
}

func newDiscardHandler(staging *fakeDuplicateStaging) *CaseHandler {
	svc := app.NewCaseService(&config.Config{}, &fakeCaseStore{}, nil, nil, staging)
	return NewCaseHandler(svc)
}

// serveDiscard 以指定的路徑參數執行 DiscardDuplicateCandidate。
func serveDiscard(h *CaseHandler, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/cases/import/duplicates/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}

	h.DiscardDuplicateCandidate(c)
	return w
}

// TestCaseHandler_DiscardDuplicateCandidate_StatusMapping 鎖住忽略端點的狀態碼契約：
// 前端依 404／409 分別顯示「已被處理」與「已被裁決」，全部落到 500 會讓待維護清單無法自癒。
func TestCaseHandler_DiscardDuplicateCandidate_StatusMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("成功刪除回 204 且不帶 body", func(t *testing.T) {
		id := uuid.New()
		h := newDiscardHandler(&fakeDuplicateStaging{
			candidate:  &app.DuplicateCandidate{ID: id, Status: "pending"},
			deleteRows: 1,
		})

		w := serveDiscard(h, id.String())

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.Bytes(), "204 回應不得帶 body")
	})

	t.Run("ID 格式錯誤回 400", func(t *testing.T) {
		h := newDiscardHandler(&fakeDuplicateStaging{})

		w := serveDiscard(h, "not-a-uuid")

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("查無暫存列回 404", func(t *testing.T) {
		h := newDiscardHandler(&fakeDuplicateStaging{candidate: nil})

		w := serveDiscard(h, uuid.New().String())

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("已被裁決回 409", func(t *testing.T) {
		id := uuid.New()
		h := newDiscardHandler(&fakeDuplicateStaging{
			candidate: &app.DuplicateCandidate{ID: id, Status: "confirmed_new"},
		})

		w := serveDiscard(h, id.String())

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("刪除時被搶先處理也回 409", func(t *testing.T) {
		id := uuid.New()
		h := newDiscardHandler(&fakeDuplicateStaging{
			candidate:  &app.DuplicateCandidate{ID: id, Status: "pending"},
			deleteRows: 0,
		})

		w := serveDiscard(h, id.String())

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}
