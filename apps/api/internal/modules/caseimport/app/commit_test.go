package app

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCaseRegistrar is a deterministic CaseRegistrar test double.
type fakeCaseRegistrar struct {
	created []NewCase
	skipped []CaseImportSkippedRow
}

func (f *fakeCaseRegistrar) CreateCase(ctx context.Context, in NewCase, actor Actor) (uuid.UUID, error) {
	f.created = append(f.created, in)
	return uuid.New(), nil
}

func (f *fakeCaseRegistrar) RecordSkipped(ctx context.Context, row CaseImportSkippedRow, actor Actor) {
	f.skipped = append(f.skipped, row)
}

// fakeTransportPreferenceWriter is a deterministic TransportPreferenceWriter test double.
type fakeTransportPreferenceWriter struct {
	calls []struct {
		caseID                                       uuid.UUID
		siteID, outboundVehicleID, inboundVehicleID  *uuid.UUID
		siteNameRaw, outboundNameRaw, inboundNameRaw string
	}
}

func (f *fakeTransportPreferenceWriter) UpsertTransportPreference(ctx context.Context, caseID uuid.UUID, siteID, outboundVehicleID, inboundVehicleID *uuid.UUID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string) error {
	f.calls = append(f.calls, struct {
		caseID                                       uuid.UUID
		siteID, outboundVehicleID, inboundVehicleID  *uuid.UUID
		siteNameRaw, outboundNameRaw, inboundNameRaw string
	}{caseID, siteID, outboundVehicleID, inboundVehicleID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw})
	return nil
}

// fakeSiteLookup resolves a fixed set of names to sites; anything else is "not found".
type fakeSiteLookup struct{ byName map[string]uuid.UUID }

func (f fakeSiteLookup) GetByName(ctx context.Context, name string) (*SiteRef, error) {
	id, ok := f.byName[name]
	if !ok {
		return nil, assertNotFoundErr
	}
	return &SiteRef{ID: id, Name: name}, nil
}

func (f fakeSiteLookup) List(ctx context.Context, region string, page, pageSize int) ([]SiteRef, error) {
	return nil, nil
}

// fakeVehicleLookup resolves a fixed set of display names; anything else is "not found".
type fakeVehicleLookup struct{ byName map[string]uuid.UUID }

func (f fakeVehicleLookup) GetByDisplayName(ctx context.Context, displayName string) (*VehicleRef, error) {
	id, ok := f.byName[displayName]
	if !ok {
		return nil, assertNotFoundErr
	}
	return &VehicleRef{ID: id}, nil
}

// fakeTxRunner runs fn directly without an actual transaction, matching the
// commit tests' need for a deterministic, dependency-free TxRunner double.
type fakeTxRunner struct{}

func (fakeTxRunner) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

var assertNotFoundErr = errCommitTestNotFound{}

type errCommitTestNotFound struct{}

func (errCommitTestNotFound) Error() string { return "not found" }

func TestCommitCases_SkipsUnflaggedDuplicateAndImportsFlaggedOne(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	svc := &ImportService{cases: registrar, prefRepo: &fakeTransportPreferenceWriter{}, txRunner: fakeTxRunner{}}

	dupID := uuid.New()
	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "未勾選重複", IsDuplicate: true, DuplicateCaseID: &dupID},
		{RowIndex: 2, Name: "已勾選重複", IsDuplicate: true, DuplicateCaseID: &dupID},
		{RowIndex: 3, Name: "非重複個案"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, map[int]bool{2: true}, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount, "第 2、3 列應成功匯入")
	require.Len(t, result.SkippedRows, 1)
	assert.Equal(t, 1, result.SkippedRows[0].RowIndex)
	assert.Contains(t, result.SkippedRows[0].Reasons, "偵測為重複個案，未勾選匯入")

	require.Len(t, registrar.created, 2)
	assert.Equal(t, "已勾選重複", registrar.created[0].Name)
	assert.Equal(t, "非重複個案", registrar.created[1].Name)
}

func TestCommitCases_CreatesCaseWhenSiteAndVehicleNamesDoNotMatch(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	prefWriter := &fakeTransportPreferenceWriter{}
	svc := &ImportService{
		cases:       registrar,
		siteRepo:    fakeSiteLookup{byName: map[string]uuid.UUID{}},
		vehicleRepo: fakeVehicleLookup{byName: map[string]uuid.UUID{}},
		prefRepo:    prefWriter,
		txRunner:    fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案甲", SiteName: "查無此單位", OutboundVehicle: "查無此車", InboundVehicle: "查無此車回"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, nil, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount, "單位/車輛比對不到仍應建立個案")
	assert.Empty(t, result.SkippedRows)
	require.Len(t, result.Warnings, 3, "單位與去回程車輛各自獨立比對不到，各附一則警示")

	require.Len(t, prefWriter.calls, 1)
	call := prefWriter.calls[0]
	assert.Nil(t, call.siteID)
	assert.Nil(t, call.outboundVehicleID)
	assert.Nil(t, call.inboundVehicleID)
	assert.Equal(t, "查無此單位", call.siteNameRaw)
	assert.Equal(t, "查無此車", call.outboundNameRaw)
	assert.Equal(t, "查無此車回", call.inboundNameRaw)
}

func TestCommitCases_ResolvesSiteAndVehicleWhenNamesMatch(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	prefWriter := &fakeTransportPreferenceWriter{}
	siteID := uuid.New()
	outboundID := uuid.New()
	svc := &ImportService{
		cases:       registrar,
		siteRepo:    fakeSiteLookup{byName: map[string]uuid.UUID{"竹南日照單位": siteID}},
		vehicleRepo: fakeVehicleLookup{byName: map[string]uuid.UUID{"竹南1車": outboundID}},
		prefRepo:    prefWriter,
		txRunner:    fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案乙", SiteName: "竹南日照單位", OutboundVehicle: "竹南1車", InboundVehicle: "查無此車回"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, nil, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	require.Len(t, result.Warnings, 1, "僅回程車比對不到，只附一則警示")

	require.Len(t, prefWriter.calls, 1)
	call := prefWriter.calls[0]
	require.NotNil(t, call.siteID)
	assert.Equal(t, siteID, *call.siteID)
	require.NotNil(t, call.outboundVehicleID)
	assert.Equal(t, outboundID, *call.outboundVehicleID)
	assert.Nil(t, call.inboundVehicleID)
	assert.Equal(t, "查無此車回", call.inboundNameRaw)
}
