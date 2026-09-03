package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
}

func (f *fakeCaseStore) List(ctx context.Context, region, status, q string, page, pageSize int, unresolvedLink, excludePending bool) ([]app.Case, int64, error) {
	return f.cases, int64(len(f.cases)), nil
}

func (f *fakeCaseStore) GetByID(ctx context.Context, id uuid.UUID) (*app.Case, error) {
	for i := range f.cases {
		if f.cases[i].ID == id {
			return &f.cases[i], nil
		}
	}
	return nil, errors.New("not found")
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
	s.ID = uuid.New()
	f.sched = s
	return nil
}

func (f *fakeCaseStore) GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*app.CaseSchedule, error) {
	return f.sched, nil
}

func (f *fakeCaseStore) GetActiveSchedulesForMonth(ctx context.Context, year, month int, region string) ([]app.ActiveCaseScheduleInfo, error) {
	return nil, nil
}

func (f *fakeCaseStore) UpsertTransportPreference(ctx context.Context, caseID uuid.UUID, siteID, outboundVehicleID, inboundVehicleID *uuid.UUID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string) error {
	return nil
}

func (f *fakeCaseStore) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	return true, nil
}

func (f *fakeCaseStore) CloseOpenSchedules(ctx context.Context, caseID uuid.UUID) error {
	return nil
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
	assert.Contains(t, body, `"nationalIdMasked"`)
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
