package app

import (
	"context"
	"errors"
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
	if in.ID != uuid.Nil {
		return in.ID, nil
	}
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
		return nil, ErrLookupNotFound
	}
	return &SiteRef{ID: id, Name: name}, nil
}

func (f fakeSiteLookup) List(ctx context.Context, page, pageSize int) ([]SiteRef, error) {
	return nil, nil
}

// fakeVehicleLookup resolves a fixed set of display names; anything else is "not found".
type fakeVehicleLookup struct{ byName map[string]uuid.UUID }

func (f fakeVehicleLookup) GetByDisplayName(ctx context.Context, displayName string) (*VehicleRef, error) {
	id, ok := f.byName[displayName]
	if !ok {
		return nil, ErrLookupNotFound
	}
	return &VehicleRef{ID: id}, nil
}

// fakeTxRunner runs fn directly without an actual transaction, matching the
// commit tests' need for a deterministic, dependency-free TxRunner double.
type fakeTxRunner struct{}

func (fakeTxRunner) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakeDuplicateCandidateStager is a deterministic DuplicateCandidateStager test double.
type fakeDuplicateCandidateStager struct {
	staged      []StageDuplicateCandidate
	alreadyKeys map[string]bool
	stageErr    error
}

func (f *fakeDuplicateCandidateStager) StageDuplicateRow(ctx context.Context, fileHash, rowKey string, in StageDuplicateCandidate) (uuid.UUID, bool, error) {
	if f.stageErr != nil {
		return uuid.Nil, false, f.stageErr
	}
	key := fileHash + ":" + rowKey
	if f.alreadyKeys != nil && f.alreadyKeys[key] {
		return uuid.Nil, true, nil
	}
	f.staged = append(f.staged, in)
	return uuid.New(), false, nil
}

func TestCommitCases_DuplicateRowsAlwaysStaged(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	stager := &fakeDuplicateCandidateStager{}
	svc := &ImportService{cases: registrar, duplicateStager: stager, prefRepo: &fakeTransportPreferenceWriter{}, txRunner: fakeTxRunner{}}

	dupID := uuid.New()
	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "疑似重複甲", IsDuplicate: true, DuplicateCaseID: &dupID},
		{RowIndex: 2, Name: "疑似重複乙", IsDuplicate: true, DuplicateCaseID: &dupID},
		{RowIndex: 3, Name: "非重複個案"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount, "只有非重複列直接建立個案")
	assert.Equal(t, 2, result.StagedDuplicateCount, "疑似重複列一律暫存待裁決，不直接建立個案")
	assert.Empty(t, result.SkippedRows)

	require.Len(t, stager.staged, 2)
	assert.Equal(t, "疑似重複甲", stager.staged[0].Name)
	assert.Equal(t, dupID, stager.staged[0].DuplicateCaseID)
	require.Len(t, registrar.created, 1)
	assert.Equal(t, "非重複個案", registrar.created[0].Name)
}

func TestCommitCases_AlreadyStagedDuplicateCountsAsAlreadyImported(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	dupID := uuid.New()
	stager := &fakeDuplicateCandidateStager{alreadyKeys: map[string]bool{":legacy:1": true}}
	svc := &ImportService{cases: registrar, duplicateStager: stager, prefRepo: &fakeTransportPreferenceWriter{}, txRunner: fakeTxRunner{}}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "重複重試列", IsDuplicate: true, DuplicateCaseID: &dupID},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 0, result.StagedDuplicateCount)
	assert.Equal(t, 1, result.AlreadyImportedCount)
	assert.Empty(t, stager.staged)
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

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

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

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

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

type errorSiteLookup struct{ err error }

func (f errorSiteLookup) GetByName(context.Context, string) (*SiteRef, error) {
	return nil, f.err
}

func (errorSiteLookup) List(context.Context, int, int) ([]SiteRef, error) {
	return nil, nil
}

type fakeCaseImportIdempotency struct {
	claimed map[string]bool
}

func (f *fakeCaseImportIdempotency) ClaimCaseImportRow(_ context.Context, fileHash, rowKey string, _ uuid.UUID) (bool, error) {
	if f.claimed == nil {
		f.claimed = map[string]bool{}
	}
	key := fileHash + ":" + rowKey
	if f.claimed[key] {
		return false, nil
	}
	f.claimed[key] = true
	return true, nil
}

func (f *fakeCaseImportIdempotency) IsCaseImportRowCommitted(_ context.Context, fileHash, rowKey string) (bool, error) {
	return f.claimed[fileHash+":"+rowKey], nil
}

func TestCommitCases_LookupDatabaseErrorFailsOnlyThatRow(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	svc := &ImportService{
		cases:    registrar,
		siteRepo: errorSiteLookup{err: errors.New("database unavailable")},
		txRunner: fakeTxRunner{},
	}
	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "單位查詢失敗", SiteName: "資料庫錯誤"},
		{RowIndex: 2, Name: "仍可匯入的個案"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, 1, result.FailedCount)
	require.Len(t, result.FailedRows, 1)
	assert.Equal(t, 1, result.FailedRows[0].RowIndex)
	assert.Len(t, registrar.created, 1)
	assert.Equal(t, "仍可匯入的個案", registrar.created[0].Name)
}

func TestCommitCases_IdempotencySkipsRepeatedFileRow(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	idempotency := &fakeCaseImportIdempotency{}
	svc := &ImportService{cases: registrar, txRunner: fakeTxRunner{}, idempotency: idempotency}
	preview := &CaseImportPreviewResult{
		FileHash: "sha256:file",
		Rows:     []CaseImportRowResult{{RowID: "Sheet-A:2", RowIndex: 2, Name: "同一列"}},
	}

	first, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})
	require.NoError(t, err)
	second, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})
	require.NoError(t, err)

	assert.Equal(t, 1, first.ImportedCount)
	assert.Equal(t, 0, second.ImportedCount)
	assert.Equal(t, 1, second.AlreadyImportedCount)
	assert.Len(t, second.SkippedRows, 1)
}

// 重傳同一份檔案時，先前建立的個案會在 re-parse 被自己判成疑似重複；
// 冪等鍵必須比重複分支先判，否則會替自己剛建的個案生出假的待裁決列。
func TestCommitCases_RepeatedFileDoesNotStageOwnCreatedCase(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	stager := &fakeDuplicateCandidateStager{}
	idempotency := &fakeCaseImportIdempotency{}
	svc := &ImportService{cases: registrar, txRunner: fakeTxRunner{}, idempotency: idempotency, duplicateStager: stager}

	firstPreview := &CaseImportPreviewResult{
		FileHash: "sha256:same-file",
		Rows:     []CaseImportRowResult{{RowID: "個案匯入範本:2", RowIndex: 2, Name: "重傳個案"}},
	}
	first, err := svc.CommitCases(context.Background(), firstPreview, Actor{ActorID: uuid.New()})
	require.NoError(t, err)
	require.Equal(t, 1, first.ImportedCount)

	// 第二次解析同一份檔案，該列已比對到剛建立的個案
	duplicateID := uuid.New()
	secondPreview := &CaseImportPreviewResult{
		FileHash: "sha256:same-file",
		Rows: []CaseImportRowResult{{
			RowID: "個案匯入範本:2", RowIndex: 2, Name: "重傳個案",
			IsDuplicate: true, DuplicateCaseID: &duplicateID,
		}},
	}
	second, err := svc.CommitCases(context.Background(), secondPreview, Actor{ActorID: uuid.New()})
	require.NoError(t, err)

	assert.Equal(t, 0, second.StagedDuplicateCount, "已匯入的列不得被暫存為疑似重複")
	assert.Equal(t, 0, second.ImportedCount)
	assert.Equal(t, 1, second.AlreadyImportedCount)
	assert.Empty(t, stager.staged, "不得呼叫待裁決暫存")
}

func TestCommitCases_BirthDateAndNationalIDInvalid_StillCreatesCase(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	svc := &ImportService{cases: registrar, prefRepo: &fakeTransportPreferenceWriter{}, txRunner: fakeTxRunner{}}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "格式待補正個案", BirthDateInvalid: true, BirthDateRaw: "民國78年怪日期", NationalIDInvalid: true, NationalID: "NOT-VALID"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount, "格式錯誤不擋列，個案仍建立並標記待補正")
	assert.Empty(t, result.SkippedRows)

	require.Len(t, registrar.created, 1)
	created := registrar.created[0]
	assert.True(t, created.AllowInvalidNationalID)
	assert.Nil(t, created.BirthDate)
	require.NotNil(t, created.BirthDateRaw)
	assert.Equal(t, "民國78年怪日期", *created.BirthDateRaw)
	assert.Equal(t, "NOT-VALID", created.NationalID)
}
