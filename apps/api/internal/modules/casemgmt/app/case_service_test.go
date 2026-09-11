package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/platform/config"
)

func testConfig() *config.Config {
	return &config.Config{
		EncryptionKey: bytes.Repeat([]byte("a"), 32),
		HMACKey:       bytes.Repeat([]byte("b"), 32),
	}
}

// fakeCaseStore is a deterministic CaseStore test double.
type fakeCaseStore struct {
	byID               map[uuid.UUID]*Case
	byHMAC             map[string]*Case
	byNameNorm         map[string][]Case
	createErr          error
	lastCreate         *Case
	deleted            map[uuid.UUID]bool
	softDeleteErr      error
	closedSchedulesFor uuid.UUID
	closeSchedulesErr  error
	relinkSiteIDs      []uuid.UUID
	relinkSiteErr      error
	// pendingSiteNames／relinkSiteByName 讓 RelinkAllPendingSites 的測試能依名稱
	// 分別控制結果，模擬「其中一筆比對失敗、其他筆仍要繼續」的情境。
	pendingSiteNames []string
	relinkSiteByName map[string][]uuid.UUID
	relinkErrByName  map[string]error
}

func newFakeCaseStore() *fakeCaseStore {
	return &fakeCaseStore{
		byID:       map[uuid.UUID]*Case{},
		byHMAC:     map[string]*Case{},
		byNameNorm: map[string][]Case{},
	}
}

func (f *fakeCaseStore) List(ctx context.Context, status, q, region string, page, pageSize int, unresolvedLink, excludePending bool) ([]Case, int64, error) {
	return nil, 0, nil
}

func (f *fakeCaseStore) GetByID(ctx context.Context, id uuid.UUID) (*Case, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, errors.New("case not found")
	}
	return c, nil
}

func (f *fakeCaseStore) GetByHMAC(ctx context.Context, hmac []byte) (*Case, error) {
	c, ok := f.byHMAC[string(hmac)]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (f *fakeCaseStore) GetByNameNormalized(ctx context.Context, nameNorm string) ([]Case, error) {
	return f.byNameNorm[nameNorm], nil
}

func (f *fakeCaseStore) Create(ctx context.Context, c *Case) error {
	f.lastCreate = c
	if f.createErr != nil {
		return f.createErr
	}
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	f.byID[c.ID] = c
	if len(c.NationalIDHMAC) > 0 {
		f.byHMAC[string(c.NationalIDHMAC)] = c
	}
	f.byNameNorm[c.NameNormalized] = append(f.byNameNorm[c.NameNormalized], *c)
	return nil
}

func (f *fakeCaseStore) Update(ctx context.Context, c *Case) error {
	f.byID[c.ID] = c
	return nil
}

func (f *fakeCaseStore) CreateSchedule(ctx context.Context, s *CaseSchedule) error {
	return nil
}

func (f *fakeCaseStore) GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*CaseSchedule, error) {
	return nil, nil
}

func (f *fakeCaseStore) GetActiveSchedulesForMonth(ctx context.Context, year, month int) ([]ActiveCaseScheduleInfo, error) {
	return nil, nil
}

func (f *fakeCaseStore) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	if f.softDeleteErr != nil {
		return false, f.softDeleteErr
	}
	if f.deleted == nil {
		f.deleted = map[uuid.UUID]bool{}
	}
	if f.deleted[id] {
		return false, nil
	}
	f.deleted[id] = true
	return true, nil
}

func (f *fakeCaseStore) CloseOpenSchedules(ctx context.Context, caseID uuid.UUID) error {
	f.closedSchedulesFor = caseID
	return f.closeSchedulesErr
}

func (f *fakeCaseStore) RelinkSiteByName(ctx context.Context, name string) ([]uuid.UUID, error) {
	if f.relinkSiteByName != nil || f.relinkErrByName != nil {
		return f.relinkSiteByName[name], f.relinkErrByName[name]
	}
	return f.relinkSiteIDs, f.relinkSiteErr
}

func (f *fakeCaseStore) RelinkCaregiverByName(ctx context.Context, name string) ([]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeCaseStore) ListPendingSiteNames(ctx context.Context) ([]string, error) {
	return f.pendingSiteNames, nil
}

func (f *fakeCaseStore) ListPendingCaregiverNames(ctx context.Context) ([]string, error) {
	return nil, nil
}

func TestCaseService_CreateCaseSchedule_ValidatesRequest(t *testing.T) {
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	base := CreateScheduleRequest{
		CaseID:             uuid.New(),
		EffectiveFrom:      from,
		EffectiveTo:        &to,
		Weekdays:           []int16{1, 3},
		TripPattern:        2,
		UnitPrice:          100,
		DistanceKM:         5,
		ServiceDurationMin: 30,
		Legs: []CreateScheduleLegItemRequest{
			{LegSeq: 1, Direction: "outbound", DepartTime: "09:00"},
			{LegSeq: 2, Direction: "inbound", DepartTime: "17:00"},
		},
	}

	tests := []struct {
		name   string
		mutate func(*CreateScheduleRequest)
		want   error
	}{
		{"星期值超出範圍", func(r *CreateScheduleRequest) { r.Weekdays = []int16{0} }, ErrInvalidScheduleWeekday},
		{"星期重複", func(r *CreateScheduleRequest) { r.Weekdays = []int16{1, 1} }, ErrInvalidScheduleWeekday},
		{"趟次序號重複", func(r *CreateScheduleRequest) { r.Legs[1].LegSeq = 1 }, ErrInvalidScheduleLegSeq},
		{"方向不合法", func(r *CreateScheduleRequest) { r.Legs[0].Direction = "return" }, ErrInvalidScheduleDirection},
		{"時間格式不合法", func(r *CreateScheduleRequest) { r.Legs[0].DepartTime = "9:00" }, ErrInvalidScheduleTime},
		{"單價不合法", func(r *CreateScheduleRequest) { r.UnitPrice = 0 }, ErrInvalidSchedulePrice},
		{"距離不合法", func(r *CreateScheduleRequest) { r.DistanceKM = -1 }, ErrInvalidScheduleDistance},
		{"服務時長不合法", func(r *CreateScheduleRequest) { r.ServiceDurationMin = 241 }, ErrInvalidScheduleDuration},
		{"有效日期區間不合法", func(r *CreateScheduleRequest) { earlier := from.AddDate(0, 0, -1); r.EffectiveTo = &earlier }, ErrInvalidScheduleDateRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base
			req.Weekdays = append([]int16(nil), base.Weekdays...)
			req.Legs = append([]CreateScheduleLegItemRequest(nil), base.Legs...)
			tt.mutate(&req)
			svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, nil, nil)

			_, err := svc.CreateCaseSchedule(context.Background(), req)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

type fakeCaseAuditWriter struct {
	entries []AuditEntry
	// err 讓測試模擬稽核寫入失敗，驗證呼叫端是否讓整筆操作一起失敗而非只留下資料異動。
	err error
}

func (f *fakeCaseAuditWriter) Write(_ context.Context, e AuditEntry) error {
	f.entries = append(f.entries, e)
	return f.err
}

type fakeCaseTransactionRunner struct {
	calls int
}

func (r *fakeCaseTransactionRunner) WithTx(ctx context.Context, fn func(context.Context) error) error {
	r.calls++
	return fn(ctx)
}

func TestCaseService_RelinkSiteByName(t *testing.T) {
	t.Run("稽核寫入失敗時回報 0 筆", func(t *testing.T) {
		store := newFakeCaseStore()
		store.relinkSiteIDs = []uuid.UUID{uuid.New(), uuid.New()}
		audit := &fakeCaseAuditWriter{err: errors.New("audit boom")}
		txRunner := &fakeCaseTransactionRunner{}
		svc := NewCaseService(testConfig(), store, audit, nil, nil, txRunner)

		n, err := svc.RelinkSiteByName(context.Background(), "測試據點", uuid.New(), "admin", "127.0.0.1", "test-agent")

		require.Error(t, err, "稽核失敗必須讓呼叫端知道，不能悄悄回報成功")
		assert.Zero(t, n)
		assert.Len(t, audit.entries, 1, "第二筆稽核失敗前就該中止，不繼續寫剩下的列")
	})

	t.Run("唯一命中時回報實際關聯筆數並帶入操作者", func(t *testing.T) {
		store := newFakeCaseStore()
		caseID := uuid.New()
		store.relinkSiteIDs = []uuid.UUID{caseID}
		audit := &fakeCaseAuditWriter{}
		actorID := uuid.New()
		svc := NewCaseService(testConfig(), store, audit, nil, nil)

		n, err := svc.RelinkSiteByName(context.Background(), "測試據點", actorID, "admin", "127.0.0.1", "test-agent")

		require.NoError(t, err)
		assert.Equal(t, 1, n)
		require.Len(t, audit.entries, 1)
		assert.Equal(t, &actorID, audit.entries[0].ActorID, "自動關聯的稽核仍須留下實際操作者，不可記成匿名 system")
		assert.Equal(t, "auto_relink_site", audit.entries[0].Action)
	})
}

func TestCaseService_RelinkAllPendingSites_ContinuesAfterOneNameFails(t *testing.T) {
	store := newFakeCaseStore()
	store.pendingSiteNames = []string{"壞據點", "好據點"}
	goodCaseID := uuid.New()
	store.relinkSiteByName = map[string][]uuid.UUID{"好據點": {goodCaseID}}
	store.relinkErrByName = map[string]error{"壞據點": errors.New("relink boom")}
	audit := &fakeCaseAuditWriter{}
	svc := NewCaseService(testConfig(), store, audit, nil, nil)

	total, err := svc.RelinkAllPendingSites(context.Background(), uuid.New(), "admin", "127.0.0.1", "test-agent")

	require.NoError(t, err, "單一名稱比對失敗只記 log，不能讓整個手動重新比對回報失敗")
	assert.Equal(t, 1, total, "失敗的那筆不計入總數，成功的那筆仍要算進去")
}

func TestCaseService_Delete(t *testing.T) {
	t.Run("查無個案回錯誤", func(t *testing.T) {
		store := newFakeCaseStore()
		svc := NewCaseService(testConfig(), store, nil, nil, nil)

		err := svc.Delete(context.Background(), uuid.New(), uuid.New(), "admin", "127.0.0.1", "test-agent")
		assert.Error(t, err)
	})

	t.Run("跨個案與排班異動使用交易邊界", func(t *testing.T) {
		store := newFakeCaseStore()
		caseID := uuid.New()
		store.byID[caseID] = &Case{ID: caseID, Name: "交易個案"}
		txRunner := &fakeCaseTransactionRunner{}
		svc := NewCaseService(testConfig(), store, nil, nil, nil, txRunner)

		err := svc.Delete(context.Background(), caseID, uuid.New(), "admin", "", "")

		require.NoError(t, err)
		assert.Equal(t, 1, txRunner.calls)
	})

	t.Run("成功刪除並收斂排班、寫入稽核", func(t *testing.T) {
		store := newFakeCaseStore()
		caseID := uuid.New()
		store.byID[caseID] = &Case{ID: caseID, Name: "測試個案"}
		audit := &fakeCaseAuditWriter{}
		svc := NewCaseService(testConfig(), store, audit, nil, nil)

		err := svc.Delete(context.Background(), caseID, uuid.New(), "admin", "127.0.0.1", "test-agent")
		require.NoError(t, err)
		assert.True(t, store.deleted[caseID])
		assert.Equal(t, caseID, store.closedSchedulesFor)
		require.Len(t, audit.entries, 1)
		assert.Equal(t, "delete", audit.entries[0].Action)
	})

	t.Run("已刪除的個案再次刪除回錯誤", func(t *testing.T) {
		store := newFakeCaseStore()
		caseID := uuid.New()
		store.byID[caseID] = &Case{ID: caseID}
		svc := NewCaseService(testConfig(), store, nil, nil, nil)

		require.NoError(t, svc.Delete(context.Background(), caseID, uuid.New(), "admin", "127.0.0.1", "test-agent"))
		err := svc.Delete(context.Background(), caseID, uuid.New(), "admin", "127.0.0.1", "test-agent")
		assert.Error(t, err)
	})
}

func TestCreateCase_OnlyNameSucceeds(t *testing.T) {
	store := newFakeCaseStore()
	svc := NewCaseService(testConfig(), store, nil, nil, nil)

	entity, err := svc.CreateCase(context.Background(), CreateCaseRequest{Name: "只填姓名"}, uuid.New(), "admin", "127.0.0.1", "test-agent")

	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, "只填姓名", entity.Name)
	assert.Nil(t, entity.NationalIDCipher)
	assert.Nil(t, entity.HomeAddress)
	assert.Equal(t, "active", entity.Status)
}

func TestCreateCase_WithCaregiverID(t *testing.T) {
	store := newFakeCaseStore()
	svc := NewCaseService(testConfig(), store, nil, nil, nil)
	cgID := uuid.New()

	entity, err := svc.CreateCase(context.Background(), CreateCaseRequest{
		Name:        "指定照護人員",
		CaregiverID: &cgID,
	}, uuid.New(), "admin", "127.0.0.1", "test-agent")

	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, "指定照護人員", entity.Name)
	assert.Equal(t, &cgID, entity.CaregiverID)

	newCgID := uuid.New()
	updated, err := svc.UpdateCase(context.Background(), entity.ID, UpdateCaseInput{
		CaregiverID: &newCgID,
	}, uuid.New(), "admin", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	assert.Equal(t, &newCgID, updated.CaregiverID)
}

func TestCaseAuditSnapshotsUseWhitelistWithoutSensitiveFields(t *testing.T) {
	caseEntity := &Case{
		ID:                uuid.New(),
		Name:              "王小明",
		NationalIDCipher:  []byte("ciphertext"),
		NationalIDHMAC:    []byte("hmac-index"),
		NationalIDMasked:  "A12***6789",
		HomeAddress:       stringPtr("新竹市測試路 1 號"),
		CareContactName:   stringPtr("王大明"),
		RegisteredAddress: stringPtr("新竹縣測試鄉"),
		Status:            "active",
	}

	raw, err := json.Marshal(newCaseAuditSnapshot(caseEntity))
	require.NoError(t, err)
	serialized := string(raw)
	assert.Contains(t, serialized, `"nameMasked":"王○明"`)
	assert.Contains(t, serialized, `"status":"active"`)
	assert.NotContains(t, serialized, "ciphertext")
	assert.NotContains(t, serialized, "hmac-index")
	assert.NotContains(t, serialized, "nationalIdCipher")
	assert.NotContains(t, serialized, "nationalIdHmac")
	assert.NotContains(t, serialized, "homeAddress")
	assert.NotContains(t, serialized, "careContactName")
	assert.NotContains(t, serialized, "registeredAddress")
}

func TestCreateCase_WritesSanitizedAuditSnapshot(t *testing.T) {
	audit := &fakeCaseAuditWriter{}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), audit, nil, nil)

	_, err := svc.CreateCase(context.Background(), CreateCaseRequest{
		Name:              "王小明",
		NationalID:        "A123456789",
		HomeAddress:       stringPtr("新竹市測試路 1 號"),
		CareContactName:   stringPtr("王大明"),
		RegisteredAddress: stringPtr("新竹縣測試鄉"),
	}, uuid.New(), "admin", "127.0.0.1", "test-agent")

	require.NoError(t, err)
	require.Len(t, audit.entries, 1)
	after, ok := audit.entries[0].AfterData.(caseAuditSnapshot)
	require.True(t, ok)
	assert.Equal(t, "王○明", after.NameMasked)
	serialized, err := json.Marshal(after)
	require.NoError(t, err)
	assert.NotContains(t, string(serialized), "A123456789")
	assert.NotContains(t, string(serialized), "homeAddress")
}

func stringPtr(value string) *string { return &value }

func TestCreateCase_DuplicateNationalIDNoLongerErrors(t *testing.T) {
	store := newFakeCaseStore()
	svc := NewCaseService(testConfig(), store, nil, nil, nil)

	first, err := svc.CreateCase(context.Background(), CreateCaseRequest{Name: "個案一", NationalID: "A202559750"}, uuid.New(), "admin", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := svc.CreateCase(context.Background(), CreateCaseRequest{Name: "個案二", NationalID: "A202559750"}, uuid.New(), "admin", "127.0.0.1", "test-agent")
	require.NoError(t, err, "身分證字號重複不再擋建立")
	require.NotNil(t, second)
	assert.NotEqual(t, first.ID, second.ID)
}

func TestCreateCase_RejectsMalformedNationalIDWhenProvided(t *testing.T) {
	store := newFakeCaseStore()
	svc := NewCaseService(testConfig(), store, nil, nil, nil)

	_, err := svc.CreateCase(context.Background(), CreateCaseRequest{Name: "格式錯誤個案", NationalID: "NOT-VALID"}, uuid.New(), "admin", "127.0.0.1", "test-agent")
	assert.Error(t, err)
}

func TestUpdateCase_NameSynchronizesNormalizedIndex(t *testing.T) {
	store := newFakeCaseStore()
	caseID := uuid.New()
	store.byID[caseID] = &Case{ID: caseID, Name: "舊姓名", NameNormalized: namenorm.Normalize("舊姓名")}
	svc := NewCaseService(testConfig(), store, nil, nil, nil)
	newName := " 劉温月妹 "

	entity, err := svc.UpdateCase(context.Background(), caseID, UpdateCaseInput{Name: &newName}, uuid.New(), "admin", "127.0.0.1", "test-agent")

	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, "劉温月妹", entity.Name)
	assert.Equal(t, namenorm.Normalize("劉温月妹"), entity.NameNormalized)
}

func TestUpdateCase_RejectsBlankName(t *testing.T) {
	store := newFakeCaseStore()
	caseID := uuid.New()
	store.byID[caseID] = &Case{ID: caseID, Name: "原姓名", NameNormalized: namenorm.Normalize("原姓名")}
	svc := NewCaseService(testConfig(), store, nil, nil, nil)
	blank := "   "

	_, err := svc.UpdateCase(context.Background(), caseID, UpdateCaseInput{Name: &blank}, uuid.New(), "admin", "127.0.0.1", "test-agent")

	assert.ErrorIs(t, err, ErrCaseNameRequired)
	assert.Equal(t, "原姓名", store.byID[caseID].Name)
}

func TestGetCaseByID_DecryptsNationalID(t *testing.T) {
	store := newFakeCaseStore()
	caseID := uuid.New()
	cipher, err := crypto.Encrypt("A123456789", testConfig().EncryptionKey)
	require.NoError(t, err)
	store.byID[caseID] = &Case{ID: caseID, NationalIDCipher: cipher}
	svc := NewCaseService(testConfig(), store, nil, nil, nil)

	entity, err := svc.GetCaseByID(context.Background(), caseID)

	require.NoError(t, err)
	assert.Equal(t, "A123456789", entity.NationalID)
}

func TestGetCaseByID_EmptyCipherYieldsEmptyNationalID(t *testing.T) {
	store := newFakeCaseStore()
	caseID := uuid.New()
	store.byID[caseID] = &Case{ID: caseID, Name: "無身分證個案"}
	svc := NewCaseService(testConfig(), store, nil, nil, nil)

	entity, err := svc.GetCaseByID(context.Background(), caseID)

	require.NoError(t, err)
	assert.Empty(t, entity.NationalID)
}

func TestRecordSkippedCaseImport_SanitizesPII(t *testing.T) {
	audit := &fakeCaseAuditWriter{}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), audit, nil, nil)

	svc.RecordSkippedCaseImport(context.Background(), CaseImportSkippedRow{
		RowIndex: 5,
		CaseName: "王小明",
		Reasons:  []string{"身分證格式錯誤"},
		RawValues: map[string]string{
			"姓名":    "王小明",
			"身分證字號": "A123456789",
			"居住地":   "新竹市測試路 1 號",
		},
	}, uuid.New(), "admin", "", "")

	require.Len(t, audit.entries, 1)
	row, ok := audit.entries[0].AfterData.(CaseImportSkippedRow)
	require.True(t, ok)
	assert.Equal(t, "王○明", row.CaseName)
	assert.Equal(t, "A12***6789", row.RawValues["身分證字號"])
	assert.Equal(t, "[REDACTED]", row.RawValues["居住地"])
}
