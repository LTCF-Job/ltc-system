package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

// fakeSiteStore is a deterministic app.SiteStore test double.
type fakeSiteStore struct {
	created   *app.Site
	updated   *app.Site
	deletedID uuid.UUID
	deleteErr error
}

func (f *fakeSiteStore) List(ctx context.Context, region, q, status string, page, pageSize int) ([]app.Site, int64, error) {
	return nil, 0, nil
}

func (f *fakeSiteStore) GetByID(ctx context.Context, id uuid.UUID) (*app.Site, error) {
	return nil, errors.New("not found")
}

func (f *fakeSiteStore) Create(ctx context.Context, s *app.Site) error {
	f.created = s
	return nil
}

func (f *fakeSiteStore) Update(ctx context.Context, s *app.Site) error {
	f.updated = s
	return nil
}

func (f *fakeSiteStore) Delete(ctx context.Context, id uuid.UUID) error {
	f.deletedID = id
	return f.deleteErr
}

func newTestSiteHandler(store *fakeSiteStore) *SiteHandler {
	return NewSiteHandler(app.NewSiteService(store))
}

func TestSiteHandler_Create_BindsDTOAndPersists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"竹北站","address":"竹北市文興路一段1號","region":"新竹縣","status":"active"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, store.created, "Create must actually call the store, not just echo the request")
	assert.Equal(t, "竹北站", store.created.Name)
	assert.Equal(t, "新竹縣", store.created.Region)
}

func TestSiteHandler_Update_PersistsChange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"竹北站(更新)","address":"竹北市文興路一段1號","region":"新竹縣","status":"active"}`
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/sites/"+id.String(), strings.NewReader(body))
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Update(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, store.updated, "Update must actually call the store, not just echo the request")
	assert.Equal(t, id, store.updated.ID)
	assert.Equal(t, "竹北站(更新)", store.updated.Name)
}

func TestSiteHandler_Delete_CallsStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/sites/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, id, store.deletedID, "Delete must actually call the store, not just return success")
}

// TestSiteHandler_Delete_UnclassifiedStoreError_Returns500 確認未被辨識的底層錯誤（如資料庫
// 連線失敗）不會被誤判為使用者輸入錯誤：只有 app.ErrSiteInUse／app.ErrSiteNotFound 才會被分流成
// 409／404，其餘一律視為系統錯誤回 500，避免把「系統壞了」講成「你填錯了」。
func TestSiteHandler_Delete_UnclassifiedStoreError_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{deleteErr: errors.New("connection reset by peer")}
	h := newTestSiteHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/sites/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestSiteHandler_Delete_SiteInUse_Returns409 確認仍被個案等資料參照（外鍵違反）的據點刪除
// 會回 409 RESOURCE_IN_USE 並附上可讀原因，而不是通用的 400 驗證失敗。
func TestSiteHandler_Delete_SiteInUse_Returns409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{deleteErr: app.ErrSiteInUse}
	h := newTestSiteHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/sites/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	require.Equal(t, http.StatusConflict, w.Code)

	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details []struct {
				Field string `json:"field"`
			} `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "RESOURCE_IN_USE", resp.Error.Code)
	assert.NotContains(t, resp.Error.Message, "id", "訊息不應殘留內部欄位代稱")
	assert.Empty(t, resp.Error.Details, "不應再帶 Field: id 這種會被前端顯示成【id】的 detail")
}

// TestSiteHandler_Delete_SiteNotFound_Returns404 確認查無此據點回 404，而不是被籠統地
// 當成使用者輸入或系統錯誤。
func TestSiteHandler_Delete_SiteNotFound_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{deleteErr: app.ErrSiteNotFound}
	h := newTestSiteHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/sites/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestSiteHandler_Create_ResponseShape 鎖定回應的 JSON 欄位契約。
func TestSiteHandler_Create_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestSiteHandler(&fakeSiteStore{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"竹北站","address":"文興路","region":"新竹縣","status":"active"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))

	for _, field := range []string{"id", "name", "address", "region", "remarks", "status", "createdAt", "updatedAt"} {
		_, ok := envelope.Data[field]
		assert.Truef(t, ok, "response must keep field %q", field)
	}
	assert.Len(t, envelope.Data, 8, "response must not gain or lose fields")
}

func TestSiteHandler_Create_WithoutStatus_DefaultsToActive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 模擬前端送出的請求：未帶 status
	body := `{"name":"竹北日照中心","address":"竹北市光明六路1號","region":"新竹縣"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, store.created)
	assert.Equal(t, "active", store.created.Status, "未提供 status 時應自動預設為 active")
}

func TestSiteHandler_Create_MissingRequiredFields_ReturnsValidationDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 故意缺少必填的 name（address 與 region 已改為選填）
	body := `{"address":"文興路"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details []struct {
				Field  string `json:"field"`
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "VALIDATION_FAILED", resp.Error.Code)
	require.NotEmpty(t, resp.Error.Details, "驗證失敗時必須回傳詳細 details 條列錯誤欄位")

	fieldMap := make(map[string]string)
	for _, d := range resp.Error.Details {
		fieldMap[d.Field] = d.Reason
	}
	assert.Contains(t, fieldMap, "name", "必須指名 name 欄位錯誤")
}

func TestSiteHandler_Create_WithoutAddressAndRegion_Succeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"無地址單位"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, store.created)
	assert.Equal(t, "無地址單位", store.created.Name)
	assert.Equal(t, "", store.created.Address)
	assert.Equal(t, "", store.created.Region)
}
