package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
	"ltc-system/apps/api/internal/platform/config"
)

// fakeCaseStore is a deterministic app.CaseStore test double.
type fakeCaseStore struct {
	cases []app.Case
	sched *app.CaseSchedule

	// listFlags 保留最後一次 List 收到的待維護篩選旗標，供 handler 的預設值測試斷言。
	listFlags caseListFlags

	// createScheduleErr 讓測試模擬 infra 層對 pg 錯誤分類後回傳的 domain sentinel
	// （ErrScheduleOverlap／ErrScheduleInvalidReference），驗證 handler 端的狀態碼映射。
	createScheduleErr error
	// createdScheduleCount 記錄 CreateSchedule 實際被呼叫（成功寫入）的次數，
	// 用於驗證驗證失敗時完全不會嘗試寫入。
	createdScheduleCount int
}

// caseListFlags 是 CaseStore.List 的兩個待維護篩選參數。
type caseListFlags struct {
	unresolvedLink bool
	excludePending bool
}

func (f *fakeCaseStore) List(ctx context.Context, status, q, region string, page, pageSize int, unresolvedLink, excludePending bool) ([]app.Case, int64, error) {
	f.listFlags = caseListFlags{unresolvedLink: unresolvedLink, excludePending: excludePending}
	return f.cases, int64(len(f.cases)), nil
}

func (f *fakeCaseStore) GetByID(ctx context.Context, id uuid.UUID) (*app.Case, error) {
	for i := range f.cases {
		if f.cases[i].ID == id {
			return &f.cases[i], nil
		}
	}
	// 比照真實 CaseRepository.GetByID：查無資料回傳 app.ErrCaseNotFound sentinel，
	// 讓 handler 的 errors.Is 判斷可以正確辨識並映射為 404。
	return nil, app.ErrCaseNotFound
}

func (f *fakeCaseStore) GetByHMAC(ctx context.Context, hmac []byte) (*app.Case, error) {
	return nil, errors.New("not found")
}

func (f *fakeCaseStore) GetByNameNormalized(ctx context.Context, nameNorm string) ([]app.Case, error) {
	return nil, nil
}

func (f *fakeCaseStore) Create(ctx context.Context, c *app.Case) error {
	c.ID = uuid.New()
	f.cases = append(f.cases, *c)
	return nil
}

func (f *fakeCaseStore) Update(ctx context.Context, c *app.Case) error {
	return nil
}

func (f *fakeCaseStore) CreateSchedule(ctx context.Context, s *app.CaseSchedule) error {
	if f.createScheduleErr != nil {
		return f.createScheduleErr
	}
	s.ID = uuid.New()
	f.sched = s
	f.createdScheduleCount++
	return nil
}

func (f *fakeCaseStore) GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*app.CaseSchedule, error) {
	return f.sched, nil
}

func (f *fakeCaseStore) GetActiveSchedulesForMonth(ctx context.Context, year, month int) ([]app.ActiveCaseScheduleInfo, error) {
	return nil, nil
}

func (f *fakeCaseStore) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	return true, nil
}

func (f *fakeCaseStore) CloseOpenSchedules(ctx context.Context, caseID uuid.UUID) error {
	return nil
}

func (f *fakeCaseStore) RelinkSiteByName(ctx context.Context, name string) ([]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeCaseStore) RelinkCaregiverByName(ctx context.Context, name string) ([]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeCaseStore) ListPendingSiteNames(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (f *fakeCaseStore) ListPendingCaregiverNames(ctx context.Context) ([]string, error) {
	return nil, nil
}

func newTestCaseHandler(store *fakeCaseStore) *CaseHandler {
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	return NewCaseHandler(svc)
}

// caseWithSecrets 是帶有加密身分證密文/HMAC 的個案樣本，用來驗證這些欄位絕不外洩。
func caseWithSecrets() app.Case {
	return app.Case{
		ID:               uuid.New(),
		Name:             "王小明",
		NationalIDCipher: []byte("secret-cipher-bytes"),
		NationalIDHMAC:   []byte("secret-hmac-bytes"),
		NationalIDMasked: "A12***4567",
		HomeAddress:      strPtr("竹北市文興路一段1號"),
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func strPtr(s string) *string { return &s }

// TestCaseHandler_List_DoesNotLeakEncryptedNationalID 鎖定回應必須排除
// NationalIDCipher／NationalIDHMAC，且欄位需為 camelCase。
func TestCaseHandler_List_DoesNotLeakEncryptedNationalID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeCaseStore{cases: []app.Case{caseWithSecrets()}}
	h := newTestCaseHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)

	h.List(c)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "secret-cipher-bytes")
	assert.NotContains(t, body, "secret-hmac-bytes")
	assert.NotContains(t, body, "NationalIDCipher")
	assert.NotContains(t, body, "NationalIDHMAC")
	assert.NotContains(t, body, "nationalIdCipher")
	assert.NotContains(t, body, "nationalIdHmac")
	assert.Contains(t, body, `"homeAddress"`)
	assert.Contains(t, body, `"nationalId"`)
}

// TestCaseHandler_Get_ResponseIsCamelCase 確認單筆查詢回應為 camelCase 契約，
// 不是 Go struct 預設的 PascalCase。
func TestCaseHandler_Get_ResponseIsCamelCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sample := caseWithSecrets()
	store := &fakeCaseStore{cases: []app.Case{sample}}
	h := newTestCaseHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+sample.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: sample.ID.String()}}

	h.Get(c)

	require.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))

	_, hasName := envelope.Data["name"]
	_, hasPascalName := envelope.Data["Name"]
	assert.True(t, hasName, "response must expose camelCase \"name\"")
	assert.False(t, hasPascalName, "response must not expose PascalCase \"Name\"")

	_, hasCaregiverID := envelope.Data["caregiverId"]
	_, hasPascalCaregiverID := envelope.Data["CaregiverID"]
	assert.True(t, hasCaregiverID, "response must expose camelCase \"caregiverId\"")
	assert.False(t, hasPascalCaregiverID, "response must not expose PascalCase \"CaregiverID\"")

	_, hasCipher := envelope.Data["nationalIdCipher"]
	_, hasHMAC := envelope.Data["nationalIdHmac"]
	assert.False(t, hasCipher, "response must not expose nationalIdCipher")
	assert.False(t, hasHMAC, "response must not expose nationalIdHmac")
}

// TestCaseHandler_GetSchedule_LegsAreCamelCase 確認排班回應（含巢狀 legs）為
// camelCase 契約。
func TestCaseHandler_GetSchedule_LegsAreCamelCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	caseID := uuid.New()
	store := &fakeCaseStore{
		sched: &app.CaseSchedule{
			ID:     uuid.New(),
			CaseID: caseID,
			Legs: []app.ScheduleLeg{
				{ID: uuid.New(), LegSeq: 1, Direction: "outbound", DepartTime: "09:40"},
			},
		},
	}
	h := newTestCaseHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID.String()+"/schedule", nil)
	c.Params = gin.Params{{Key: "id", Value: caseID.String()}}

	h.GetSchedule(c)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"legSeq"`)
	assert.Contains(t, body, `"departTime"`)
	assert.NotContains(t, body, `"LegSeq"`)
	assert.NotContains(t, body, `"DepartTime"`)
}

// TestCaseHandler_GetSchedule_NoActiveSchedule_ReturnsNullNotNotFound 確認個案
// 尚無現行排班時回傳 200 搭配 data:null，而非 404，因為這是合法的空狀態而非錯誤。
func TestCaseHandler_GetSchedule_NoActiveSchedule_ReturnsNullNotNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	caseID := uuid.New()
	store := &fakeCaseStore{}
	h := newTestCaseHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID.String()+"/schedule", nil)
	c.Params = gin.Params{{Key: "id", Value: caseID.String()}}

	h.GetSchedule(c)

	require.Equal(t, http.StatusOK, w.Code)
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	assert.Equal(t, "null", string(envelope.Data))
}

// fakeRelinker 讓測試控制 RelinkByName 回傳的筆數與呼叫參數。
type fakeRelinker struct {
	n         int
	err       error
	calledFor string
}

func (f *fakeRelinker) RelinkByName(ctx context.Context, name string, actorID uuid.UUID, actorRole, ip, ua string) (int, error) {
	f.calledFor = name
	return f.n, f.err
}

func TestCaseHandler_Create_NilRelinkerOmitsMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeCaseStore{}
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	h := NewCaseHandler(svc) // 未傳入 relinker

	body := `{"name":"陳大華","siteId":"` + uuid.New().String() + `","caregiverId":"` + uuid.New().String() + `"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cases", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Meta map[string]any `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Meta, "relinker 未接線時不應出現 pendingRelinked")
}

func TestCaseHandler_Create_RelinkerReportsPendingRelinkedCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeCaseStore{}
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	relinker := &fakeRelinker{n: 3}
	h := NewCaseHandler(svc, relinker)

	body := `{"name":"陳大華","siteId":"` + uuid.New().String() + `","caregiverId":"` + uuid.New().String() + `"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cases", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "陳大華", relinker.calledFor)
	var resp struct {
		Meta struct {
			PendingRelinked int `json:"pendingRelinked"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 3, resp.Meta.PendingRelinked)
}

func TestCaseHandler_SaveSchedule_UsesPathCaseID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pathCaseID := uuid.New()
	store := &fakeCaseStore{cases: []app.Case{{ID: pathCaseID}}}
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	h := NewCaseHandler(svc)

	body := `{"effectiveFrom":"2026-09-01T00:00:00Z","weekdays":[1,2,3,4,5],"tripPattern":1,"unitPrice":115,"distanceKm":5,"serviceDurationMin":10,"serviceCode":"BD03","legs":[{"legSeq":1,"direction":"outbound","departTime":"09:00"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/cases/"+pathCaseID.String()+"/schedule", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: pathCaseID.String()}}

	h.SaveSchedule(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, store.sched)
	assert.Equal(t, pathCaseID, store.sched.CaseID)
}

// TestCaseHandler_SaveSchedule_CaseNotFound_Returns404NotGenericBadRequest 確認排班的
// 個案 ID 若查無此個案，回傳 404 查無資料而非把「個案不存在」偽裝成 400 輸入格式錯誤。
func TestCaseHandler_SaveSchedule_CaseNotFound_Returns404NotGenericBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	missingCaseID := uuid.New()
	store := &fakeCaseStore{} // 空清單：GetByID 一定找不到
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	h := NewCaseHandler(svc)

	body := `{"effectiveFrom":"2026-09-01T00:00:00Z","weekdays":[1,2,3,4,5],"tripPattern":1,"unitPrice":115,"distanceKm":5,"serviceDurationMin":10,"serviceCode":"BD03","legs":[{"legSeq":1,"direction":"outbound","departTime":"09:00"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/cases/"+missingCaseID.String()+"/schedule", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: missingCaseID.String()}}

	h.SaveSchedule(c)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"NOT_FOUND"`)
}

// TestCaseHandler_CreateSchedule_InvalidWeekday_ReturnsSpecificChineseReason 確認排班驗證
// 失敗時（星期設定不正確）details 附上具體中文原因，而不是只有通用的「輸入資料不符合規則」。
func TestCaseHandler_CreateSchedule_InvalidWeekday_ReturnsSpecificChineseReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	caseID := uuid.New()
	store := &fakeCaseStore{cases: []app.Case{{ID: caseID}}}
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	h := NewCaseHandler(svc)

	body := `{"caseId":"` + caseID.String() + `","effectiveFrom":"2026-09-01T00:00:00Z","weekdays":[0],"tripPattern":1,"unitPrice":115,"distanceKm":5,"serviceDurationMin":10,"serviceCode":"BD03","legs":[{"legSeq":1,"direction":"outbound","departTime":"09:00"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cases/schedules", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateSchedule(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	body2 := w.Body.String()
	assert.Contains(t, body2, "星期設定不正確")
	assert.NotContains(t, body2, "weekdays must be")
}

// TestCaseHandler_CreateSchedule_OmittedUnitPrice_GetsSpecificBusinessReason 確認省略
// unitPrice 時仍會被擋下、且完全不會寫入，訊息是 validateScheduleRequest 給出的具體業務
// 原因（單價必須大於 0），而不是 binding 層「unitPrice為必填項目」這種不夠具體的訊息——
// 這正是 DTO 特意不用 binding:"required" 的原因：把「未填」與「填 0」都交給同一段業務
// 規則判斷並給出一致、正確的中文原因。
func TestCaseHandler_CreateSchedule_OmittedUnitPrice_GetsSpecificBusinessReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	caseID := uuid.New()
	store := &fakeCaseStore{cases: []app.Case{{ID: caseID}}}
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	h := NewCaseHandler(svc)

	body := `{"caseId":"` + caseID.String() + `","effectiveFrom":"2026-09-01T00:00:00Z","weekdays":[1],"tripPattern":1,"distanceKm":5,"serviceDurationMin":10,"serviceCode":"BD03","legs":[{"legSeq":1,"direction":"outbound","departTime":"09:00"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cases/schedules", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateSchedule(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "單價必須大於 0")
	assert.Zero(t, store.createdScheduleCount, "省略必填欄位時不應寫入資料")
}

// TestCaseHandler_CreateSchedule_ExplicitZeroUnitPrice_GetsSameBusinessReason 確認明確
// 填 0 與省略欄位得到一致的具體業務原因，而不是被 binding 層攔成「為必填項目」——單價與
// 距離依業務規則本來就不允許 0（見 validateScheduleRequest），這裡驗證的是訊息的具體性，
// 不是 0 應該被接受。
func TestCaseHandler_CreateSchedule_ExplicitZeroUnitPrice_GetsSameBusinessReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	caseID := uuid.New()
	store := &fakeCaseStore{cases: []app.Case{{ID: caseID}}}
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	h := NewCaseHandler(svc)

	body := `{"caseId":"` + caseID.String() + `","effectiveFrom":"2026-09-01T00:00:00Z","weekdays":[1],"tripPattern":1,"unitPrice":0,"distanceKm":5,"serviceDurationMin":10,"serviceCode":"BD03","legs":[{"legSeq":1,"direction":"outbound","departTime":"09:00"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cases/schedules", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateSchedule(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "單價必須大於 0")
	assert.NotContains(t, w.Body.String(), "為必填項目")
	assert.Zero(t, store.createdScheduleCount)
}

// TestCaseHandler_CreateSchedule_OverlapConstraintViolation_Returns409NotGeneric 確認
// infra 層的 pg 23P01（no_overlapping_case_schedule）會經 ErrScheduleOverlap 轉譯成 409
// ASSIGNMENT_OVERLAP，而非未分類的 500 或偽裝成 400 使用者輸入錯誤。
func TestCaseHandler_CreateSchedule_OverlapConstraintViolation_Returns409NotGeneric(t *testing.T) {
	gin.SetMode(gin.TestMode)
	caseID := uuid.New()
	store := &fakeCaseStore{cases: []app.Case{{ID: caseID}}, createScheduleErr: app.ErrScheduleOverlap}
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	h := NewCaseHandler(svc)

	body := `{"caseId":"` + caseID.String() + `","effectiveFrom":"2026-09-01T00:00:00Z","weekdays":[1],"tripPattern":1,"unitPrice":115,"distanceKm":5,"serviceDurationMin":10,"serviceCode":"BD03","legs":[{"legSeq":1,"direction":"outbound","departTime":"09:00"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cases/schedules", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateSchedule(c)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "ASSIGNMENT_OVERLAP")
}

// TestCaseHandler_CreateSchedule_InvalidReference_Returns400WithReason 確認 pg 23503
// （選到不存在的個案/車輛）經 ErrScheduleInvalidReference 轉譯成 400 並附具體中文原因，
// 而非未分類的 500。
func TestCaseHandler_CreateSchedule_InvalidReference_Returns400WithReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	caseID := uuid.New()
	store := &fakeCaseStore{cases: []app.Case{{ID: caseID}}, createScheduleErr: app.ErrScheduleInvalidReference}
	svc := app.NewCaseService(&config.Config{}, store, nil, nil, nil)
	h := NewCaseHandler(svc)

	body := `{"caseId":"` + caseID.String() + `","effectiveFrom":"2026-09-01T00:00:00Z","weekdays":[1],"tripPattern":1,"unitPrice":115,"distanceKm":5,"serviceDurationMin":10,"serviceCode":"BD03","legs":[{"legSeq":1,"direction":"outbound","departTime":"09:00"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cases/schedules", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateSchedule(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "所選擇的車輛或個案資料不存在")
}

// TestCaseHandler_Create_MissingName_ReturnsSpecificReason 確認建立個案缺姓名時給出
// 「請輸入個案姓名」的具體原因，而非全部退回通用驗證失敗句。
func TestCaseHandler_Create_MissingName_ReturnsSpecificReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeCaseStore{}
	h := newTestCaseHandler(store)

	body := `{"name":"  ","siteId":"` + uuid.New().String() + `","caregiverId":"` + uuid.New().String() + `","status":"active"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cases", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "請輸入個案姓名")
}
