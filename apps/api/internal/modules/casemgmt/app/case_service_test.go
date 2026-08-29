package app

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	byID           map[uuid.UUID]*Case
	byHMAC         map[string]*Case
	byNameNorm     map[string][]Case
	createErr      error
	lastCreate     *Case
	lastUpsertPref struct {
		caseID                                                     uuid.UUID
		siteID, outboundVehicleID, inboundVehicleID                *uuid.UUID
		siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string
	}
}

func newFakeCaseStore() *fakeCaseStore {
	return &fakeCaseStore{
		byID:       map[uuid.UUID]*Case{},
		byHMAC:     map[string]*Case{},
		byNameNorm: map[string][]Case{},
	}
}

func (f *fakeCaseStore) List(ctx context.Context, region, status, q string, page, pageSize int, unresolvedLink bool) ([]Case, int64, error) {
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

func (f *fakeCaseStore) GetActiveSchedulesForMonth(ctx context.Context, year, month int, region string) ([]ActiveCaseScheduleInfo, error) {
	return nil, nil
}

func (f *fakeCaseStore) UpsertTransportPreference(ctx context.Context, caseID uuid.UUID, siteID, outboundVehicleID, inboundVehicleID *uuid.UUID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string) error {
	f.lastUpsertPref.caseID = caseID
	f.lastUpsertPref.siteID = siteID
	f.lastUpsertPref.outboundVehicleID = outboundVehicleID
	f.lastUpsertPref.inboundVehicleID = inboundVehicleID
	f.lastUpsertPref.siteNameRaw = siteNameRaw
	f.lastUpsertPref.outboundVehicleNameRaw = outboundVehicleNameRaw
	f.lastUpsertPref.inboundVehicleNameRaw = inboundVehicleNameRaw
	return nil
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
	assert.Nil(t, entity.Region)
	assert.Nil(t, entity.ClaimStartDate)
	assert.Equal(t, "active", entity.Status)
}

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

func TestUpdateCaseTransportPreference_PartialUpdateKeepsOtherIDsIntact(t *testing.T) {
	store := newFakeCaseStore()
	svc := NewCaseService(testConfig(), store, nil, nil, nil)
	caseID := uuid.New()
	store.byID[caseID] = &Case{ID: caseID}

	siteID := uuid.New()
	_, err := svc.UpdateCaseTransportPreference(context.Background(), caseID, &siteID, nil, nil, "", "未比對到的去程車", "未比對到的回程車")

	require.NoError(t, err)
	assert.Equal(t, &siteID, store.lastUpsertPref.siteID)
	assert.Nil(t, store.lastUpsertPref.outboundVehicleID, "未提供的去程車 ID 應維持 nil，交由 repo 端 COALESCE 保留現況")
	assert.Nil(t, store.lastUpsertPref.inboundVehicleID)
	assert.Equal(t, "未比對到的去程車", store.lastUpsertPref.outboundVehicleNameRaw)
	assert.Equal(t, "未比對到的回程車", store.lastUpsertPref.inboundVehicleNameRaw)
}
