package app_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/config"
)

// --- 測試替身 ---

type fakeSourceReader struct {
	sources []app.GovClaimSource
	err     error

	gotStart   time.Time
	gotEnd     time.Time
	gotRegion  string
	gotCaseIDs []uuid.UUID
}

func (f *fakeSourceReader) QueryGovClaimSources(_ context.Context, scope app.ClaimScope) ([]app.GovClaimSource, error) {
	f.gotStart, f.gotEnd, f.gotRegion, f.gotCaseIDs = scope.StartDate, scope.EndDate, scope.RegionValue(), scope.CaseIDs
	return f.sources, f.err
}

type fakeExportStore struct {
	created    []app.ExportJobCreate
	completed  []app.GovClaimCaseFile
	lines      []app.ExportLine
	failed     []string
	jobID      uuid.UUID
	storedJob  app.GovClaimJob
	ciphers    app.NationalIDCiphers
	caseLines  map[uuid.UUID][]app.ExportLine
	createFail error
}

func (f *fakeExportStore) CreateJob(_ context.Context, job app.ExportJobCreate) (uuid.UUID, error) {
	if f.createFail != nil {
		return uuid.Nil, f.createFail
	}
	f.created = append(f.created, job)
	return f.jobID, nil
}

func (f *fakeExportStore) CompleteJob(_ context.Context, _ uuid.UUID, files []app.GovClaimCaseFile, lines []app.ExportLine) error {
	f.completed = files
	f.lines = lines

	f.storedJob.Files = files
	f.storedJob.TotalCases = len(files)
	f.storedJob.TotalRows = 0
	for _, file := range files {
		f.storedJob.TotalRows += file.RowCount
	}
	f.caseLines = make(map[uuid.UUID][]app.ExportLine)
	for _, line := range lines {
		f.caseLines[line.CaseID] = append(f.caseLines[line.CaseID], line)
	}
	return nil
}

func (f *fakeExportStore) FailJob(_ context.Context, _ uuid.UUID, message string) error {
	f.failed = append(f.failed, message)
	return nil
}

func (f *fakeExportStore) GetJob(_ context.Context, _ uuid.UUID) (app.GovClaimJob, error) {
	return f.storedJob, nil
}

func (f *fakeExportStore) ListJobs(_ context.Context, _, _ int) ([]app.GovClaimJob, int64, error) {
	return []app.GovClaimJob{f.storedJob}, 1, nil
}

func (f *fakeExportStore) LoadCaseLines(_ context.Context, _, caseID uuid.UUID) ([]app.ExportLine, error) {
	return f.caseLines[caseID], nil
}

func (f *fakeExportStore) LoadNationalIDCiphers(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (app.NationalIDCiphers, error) {
	return f.ciphers, nil
}

type recordingRenderer struct {
	batches [][]govform.ClaimRow
}

func (r *recordingRenderer) RenderTripSummary(string, []app.TripSummaryVehicle) ([]byte, error) {
	return nil, nil
}

func (r *recordingRenderer) RenderHsinchuSchedule([]app.HsinchuScheduleItem, []app.HsinchuScheduleItem) ([]byte, error) {
	return nil, nil
}

func (r *recordingRenderer) RenderGovClaim(rows []govform.ClaimRow) ([]byte, error) {
	copied := make([]govform.ClaimRow, len(rows))
	copy(copied, rows)
	r.batches = append(r.batches, copied)
	return []byte("PK-workbook-" + string(rune('A'+len(r.batches)))), nil
}

type recordingArchiver struct {
	entries []app.ZipEntry
}

func (a *recordingArchiver) BuildZip(entries []app.ZipEntry) ([]byte, error) {
	a.entries = entries
	return []byte("PK-zip"), nil
}

type discardAuditWriter struct{}

func (discardAuditWriter) Write(context.Context, app.AuditEntry) error { return nil }

type failingAuditWriter struct{ err error }

func (w failingAuditWriter) Write(context.Context, app.AuditEntry) error { return w.err }

type immutableExportStore struct {
	*fakeExportStore
	content []byte
	err     error
}

func (s *immutableExportStore) LoadExportFile(context.Context, uuid.UUID, uuid.UUID) ([]byte, error) {
	return s.content, s.err
}

type stubPrecheckRepo struct {
	incomplete []app.IncompleteCase
}

func (s stubPrecheckRepo) FindIncompleteActiveCases(context.Context, app.ClaimScope) ([]app.IncompleteCase, error) {
	return s.incomplete, nil
}

func (s stubPrecheckRepo) FindUnresolvedConflicts(context.Context, app.ClaimScope) ([]app.UnresolvedConflict, error) {
	return nil, nil
}

type recordingPrecheckRepo struct {
	scope app.ClaimScope
}

func (r *recordingPrecheckRepo) FindIncompleteActiveCases(_ context.Context, scope app.ClaimScope) ([]app.IncompleteCase, error) {
	r.scope = scope
	return nil, nil
}

func (r *recordingPrecheckRepo) FindUnresolvedConflicts(_ context.Context, scope app.ClaimScope) ([]app.UnresolvedConflict, error) {
	r.scope = scope
	return nil, nil
}

// --- 測試工具 ---

var testEncryptionKey = mustDecodeKey("MDEwMjAzMDQwNTA2MDcwODAxMDIwMzA0MDUwNjA3MDg=")

func mustDecodeKey(b64 string) []byte {
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(err)
	}
	return key
}

func mustEncrypt(t *testing.T, plain string) []byte {
	t.Helper()
	cipher, err := crypto.Encrypt(plain, testEncryptionKey)
	require.NoError(t, err)
	return cipher
}

func strPtr(s string) *string { return &s }
func intPtr(v int) *int       { return &v }

type sourceOption func(*app.GovClaimSource)

// newSource 產生一筆完整、可通過驗證的來源資料，個別測試再以 option 打壞想驗的欄位。
func newSource(t *testing.T, caseID uuid.UUID, caseCode, caseName string, driverID uuid.UUID, day int, legSeq int16, direction, departTime string, opts ...sourceOption) app.GovClaimSource {
	t.Helper()
	s := app.GovClaimSource{
		CaseID:               caseID,
		CaseName:             caseName,
		Region:               "hsinchu",
		CaseNationalIDCipher: mustEncrypt(t, "A202559750"),
		CaseNationalIDMasked: "A2****9750",
		HomeAddress:          "新竹縣竹北市光明六路264號",
		ServiceCategory:      intPtr(1),
		ServiceUsageType:     intPtr(2),
		ServiceDate:          time.Date(2026, 7, day, 0, 0, 0, 0, time.UTC),
		LegSeq:               legSeq,
		Direction:            strPtr(direction),
		DepartTime:           strPtr(departTime),
		DurationMin:          intPtr(10),
		ServiceCode:          "BD03",
		UnitPrice:            115,
		DistanceKM:           5,
		SiteAddress:          "新竹縣竹北市中正西路100號",
		PlateNo:              "BZG-7915",
		DriverID:             &driverID,
	}
	s.DriverNationalIDCipher = mustEncrypt(t, "K120098177")
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

func newService(reader app.GovClaimSourceReader, store app.ExportJobStore, renderer app.Renderer, archiver app.Archiver, precheckRepo app.PrecheckRepositoryPort) *app.GovClaimService {
	return newServiceWithAudit(reader, store, renderer, archiver, precheckRepo, discardAuditWriter{})
}

func newServiceWithAudit(reader app.GovClaimSourceReader, store app.ExportJobStore, renderer app.Renderer, archiver app.Archiver, precheckRepo app.PrecheckRepositoryPort, audit app.AuditWriter) *app.GovClaimService {
	cfg := &config.Config{EncryptionKey: testEncryptionKey}
	return app.NewGovClaimService(cfg, reader, store, renderer, archiver, app.NewPrecheckService(precheckRepo), audit)
}

func newInput(mode app.GovClaimMode, caseIDs ...uuid.UUID) app.CreateGovClaimInput {
	return app.CreateGovClaimInput{
		PeriodYM:  "11507",
		Region:    "hsinchu",
		CaseIDs:   caseIDs,
		Mode:      mode,
		CreatedBy: uuid.New(),
	}
}

// --- 測試 ---

func TestCreateGovClaimJob_OneFilePerCase(t *testing.T) {
	caseA, caseB := uuid.New(), uuid.New()
	driver := uuid.New()
	reader := &fakeSourceReader{sources: []app.GovClaimSource{
		newSource(t, caseA, "C001", "蔡曾切", driver, 1, 1, "outbound", "09:40"),
		newSource(t, caseA, "C001", "蔡曾切", driver, 1, 2, "inbound", "12:20"),
		newSource(t, caseB, "C002", "林大明", driver, 1, 1, "outbound", "09:40"),
	}}
	store := &fakeExportStore{jobID: uuid.New()}
	renderer := &recordingRenderer{}

	job, err := newService(reader, store, renderer, &recordingArchiver{}, stubPrecheckRepo{}).
		CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseA, caseB))
	require.NoError(t, err)

	assert.Len(t, store.completed, 2)
	assert.Equal(t, 2, job.TotalCases)
	assert.Equal(t, 3, job.TotalRows)
	assert.Equal(t, "林大明11507.xlsx", store.completed[0].FileName)
	assert.Equal(t, "蔡曾切11507.xlsx", store.completed[1].FileName)
	assert.Equal(t, 1, store.completed[0].RowCount)
	assert.Equal(t, 2, store.completed[1].RowCount)
	assert.Empty(t, job.Skipped)

	// 查詢期間須為該民國月份的西元起訖（左閉右開）
	assert.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), reader.gotStart)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), reader.gotEnd)
	assert.Equal(t, "xlsx", store.created[0].Format)
}

func TestCreateGovClaimJob_UsesSameClaimScopeForPrecheckAndExport(t *testing.T) {
	caseID := uuid.New()
	driverID := uuid.New()
	reader := &fakeSourceReader{sources: []app.GovClaimSource{
		newSource(t, caseID, "C001", "蔡曾切", driverID, 1, 1, "outbound", "09:40"),
	}}
	precheck := &recordingPrecheckRepo{}
	store := &fakeExportStore{jobID: uuid.New()}

	_, err := newService(reader, store, &recordingRenderer{}, &recordingArchiver{}, precheck).
		CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseID))

	require.NoError(t, err)
	assert.Equal(t, reader.gotStart, precheck.scope.StartDate)
	assert.Equal(t, reader.gotEnd, precheck.scope.EndDate)
	assert.Equal(t, reader.gotRegion, precheck.scope.RegionValue())
	assert.Equal(t, reader.gotCaseIDs, precheck.scope.CaseIDs)
}

func TestCreateGovClaimJob_RowOrderMatchesGovernmentSample(t *testing.T) {
	caseA := uuid.New()
	driver := uuid.New()
	sources := []app.GovClaimSource{
		newSource(t, caseA, "C001", "蔡曾切", driver, 2, 2, "inbound", "12:20"),
		newSource(t, caseA, "C001", "蔡曾切", driver, 1, 2, "inbound", "12:20"),
		newSource(t, caseA, "C001", "蔡曾切", driver, 2, 1, "outbound", "09:40"),
		newSource(t, caseA, "C001", "蔡曾切", driver, 1, 1, "outbound", "09:40"),
	}
	renderer := &recordingRenderer{}

	_, err := newService(&fakeSourceReader{sources: sources}, &fakeExportStore{jobID: uuid.New()}, renderer, &recordingArchiver{}, stubPrecheckRepo{}).
		CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseA))
	require.NoError(t, err)

	require.Len(t, renderer.batches, 1)
	rows := renderer.batches[0]
	require.Len(t, rows, 4)

	// leg1 整月在前、leg2 整月在後，各自依日期升冪；與範本 蔡曾切11507.xls 的列序一致
	assert.Equal(t, 1150701, rows[0].Cells[1])
	assert.Equal(t, 1150702, rows[1].Cells[1])
	assert.Equal(t, 1150701, rows[2].Cells[1])
	assert.Equal(t, 1150702, rows[3].Cells[1])
	assert.Equal(t, "outbound", rows[0].Direction)
	assert.Equal(t, "inbound", rows[2].Direction)
}

func TestCreateGovClaimJob_SnapshotOmitsPlainNationalIDs(t *testing.T) {
	caseA := uuid.New()
	driver := uuid.New()
	store := &fakeExportStore{jobID: uuid.New()}

	_, err := newService(
		&fakeSourceReader{sources: []app.GovClaimSource{newSource(t, caseA, "C001", "蔡曾切", driver, 1, 1, "outbound", "09:40")}},
		store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{},
	).CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseA))
	require.NoError(t, err)

	require.Len(t, store.lines, 1)
	line := store.lines[0]
	assert.Equal(t, "", line.Payload.Cells[0], "個案身分證明文不得寫進快照")
	assert.Equal(t, "", line.Payload.Cells[6], "服務人員身分證明文不得寫進快照")
	require.NotNil(t, line.Payload.DriverID)
	assert.Equal(t, driver, *line.Payload.DriverID)
	assert.Equal(t, "A2****9750", line.NationalIDMasked)
	assert.Equal(t, 1150701, line.ServiceDateROC)
	assert.Equal(t, 1, line.LineNo)
}

func TestCreateGovClaimJob_RejectsAuditFailureBeforeCompletingExport(t *testing.T) {
	caseID := uuid.New()
	driverID := uuid.New()
	auditErr := errors.New("audit store unavailable")
	store := &fakeExportStore{jobID: uuid.New()}

	_, err := newServiceWithAudit(
		&fakeSourceReader{sources: []app.GovClaimSource{
			newSource(t, caseID, "C001", "蔡曾切", driverID, 1, 1, "outbound", "09:40"),
		}},
		store,
		&recordingRenderer{},
		&recordingArchiver{},
		stubPrecheckRepo{},
		failingAuditWriter{err: auditErr},
	).CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseID))

	require.Error(t, err)
	assert.ErrorIs(t, err, auditErr)
	assert.Contains(t, err.Error(), "write export audit")
	assert.Empty(t, store.completed, "稽核失敗時不可把工作標為成功")
	assert.Len(t, store.failed, 1, "稽核失敗時應把工作標為失敗")
}

func TestCreateGovClaimJob_BlocksIncompleteSources(t *testing.T) {
	driver := uuid.New()
	tests := []struct {
		name   string
		mutate sourceOption
		reason string
	}{
		{"排班趟次對不到", func(s *app.GovClaimSource) { s.Direction = nil }, "NO_SCHEDULE_LEG"},
		{"沒有出發時間", func(s *app.GovClaimSource) { s.DepartTime = nil }, "NO_DEPART_TIME"},
		{"沒有服務時長", func(s *app.GovClaimSource) { s.DurationMin = intPtr(0) }, "NO_DEPART_TIME"},
		{"沒有司機", func(s *app.GovClaimSource) { s.DriverID = nil }, "NO_DRIVER"},
		{"服務類別未設定", func(s *app.GovClaimSource) { s.ServiceCategory = nil }, "NO_SERVICE_CATEGORY"},
		{"服務類別超出範圍", func(s *app.GovClaimSource) { s.ServiceCategory = intPtr(9) }, "NO_SERVICE_CATEGORY"},
		{"服務使用類型未設定", func(s *app.GovClaimSource) { s.ServiceUsageType = nil }, "NO_SERVICE_USAGE_TYPE"},
		{"服務使用類型超出範圍", func(s *app.GovClaimSource) { s.ServiceUsageType = intPtr(0) }, "NO_SERVICE_USAGE_TYPE"},
		{"單價為零", func(s *app.GovClaimSource) { s.UnitPrice = 0 }, "NO_UNIT_PRICE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caseA := uuid.New()
			good := newSource(t, caseA, "C001", "蔡曾切", driver, 1, 1, "outbound", "09:40")
			bad := newSource(t, caseA, "C001", "蔡曾切", driver, 2, 1, "outbound", "09:40", tt.mutate)
			store := &fakeExportStore{jobID: uuid.New()}

			_, err := newService(&fakeSourceReader{sources: []app.GovClaimSource{good, bad}}, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{}).
				CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseA))
			assert.ErrorIs(t, err, app.ErrInvalidClaimData)
			assert.Empty(t, store.completed, "存在非法列時不可產生部分申報檔")
			assert.Len(t, store.failed, 1)
			assert.Equal(t, "申報資料不完整，請修正後重新匯出", store.failed[0])
		})
	}
}

func TestCreateGovClaimJob_CaseWithoutUsableRowsProducesNoFile(t *testing.T) {
	caseA := uuid.New()
	driver := uuid.New()
	source := newSource(t, caseA, "C001", "蔡曾切", driver, 1, 1, "outbound", "09:40", func(s *app.GovClaimSource) { s.Direction = nil })
	store := &fakeExportStore{jobID: uuid.New()}

	_, err := newService(&fakeSourceReader{sources: []app.GovClaimSource{source}}, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{}).
		CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseA))

	assert.ErrorIs(t, err, app.ErrInvalidClaimData)
	assert.Empty(t, store.completed)
	assert.Len(t, store.failed, 1, "產不出檔案仍要留下失敗紀錄")
}

func TestCreateGovClaimJob_PrecheckErrorBlocksJobCreation(t *testing.T) {
	caseA := uuid.New()
	store := &fakeExportStore{jobID: uuid.New()}
	precheck := stubPrecheckRepo{incomplete: []app.IncompleteCase{{ID: caseA, Name: "蔡曾切"}}}

	_, err := newService(&fakeSourceReader{}, store, &recordingRenderer{}, &recordingArchiver{}, precheck).
		CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseA))

	assert.ErrorIs(t, err, app.ErrPrecheckBlocked)
	assert.Empty(t, store.created, "檢核未過不得建立匯出工作")
}

func TestCreateGovClaimJob_InvalidPeriod(t *testing.T) {
	store := &fakeExportStore{jobID: uuid.New()}
	input := newInput(app.GovClaimModeDirect, uuid.New())
	input.PeriodYM = "2026-07"

	_, err := newService(&fakeSourceReader{}, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{}).
		CreateGovClaimJob(context.Background(), input)

	assert.ErrorIs(t, err, app.ErrInvalidPeriodYM)
	assert.Empty(t, store.created)
}

func TestCreateGovClaimJob_ZipModeRecordsZipFormat(t *testing.T) {
	caseA, caseB := uuid.New(), uuid.New()
	driver := uuid.New()
	store := &fakeExportStore{jobID: uuid.New()}
	store.storedJob.Mode = app.GovClaimModeZip
	store.storedJob.Region = "hsinchu"
	store.storedJob.PeriodYM = "11507"

	_, err := newService(&fakeSourceReader{sources: []app.GovClaimSource{
		newSource(t, caseA, "C001", "蔡曾切", driver, 1, 1, "outbound", "09:40"),
		newSource(t, caseB, "C002", "林大明", driver, 1, 1, "outbound", "09:40"),
	}}, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{}).
		CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeZip, caseA, caseB))
	require.NoError(t, err)

	assert.Equal(t, "zip", store.created[0].Format)
}

func TestRenderZip_PacksEveryCaseFile(t *testing.T) {
	caseA, caseB := uuid.New(), uuid.New()
	driver := uuid.New()
	store := &fakeExportStore{jobID: uuid.New()}
	store.storedJob.Mode = app.GovClaimModeZip
	store.storedJob.Region = "hsinchu"
	store.storedJob.PeriodYM = "11507"
	store.ciphers = app.NationalIDCiphers{
		Case:    mustEncrypt(t, "A202559750"),
		Drivers: map[uuid.UUID][]byte{driver: mustEncrypt(t, "K120098177")},
	}
	archiver := &recordingArchiver{}
	svc := newService(&fakeSourceReader{sources: []app.GovClaimSource{
		newSource(t, caseA, "C001", "蔡曾切", driver, 1, 1, "outbound", "09:40"),
		newSource(t, caseB, "C002", "林大明", driver, 1, 1, "outbound", "09:40"),
	}}, store, &recordingRenderer{}, archiver, stubPrecheckRepo{})

	job, err := svc.CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeZip, caseA, caseB))
	require.NoError(t, err)

	fileName, archive, err := svc.RenderZip(context.Background(), job.ID)
	require.NoError(t, err)
	assert.Equal(t, "gov-claim-hsinchu-11507.zip", fileName)
	assert.NotEmpty(t, archive)
	require.Len(t, archiver.entries, 2)
	assert.Equal(t, "林大明11507.xlsx", archiver.entries[0].Name)
	assert.Equal(t, "蔡曾切11507.xlsx", archiver.entries[1].Name)
}

func TestRenderZip_RejectsDirectModeJob(t *testing.T) {
	store := &fakeExportStore{jobID: uuid.New()}
	store.storedJob.Mode = app.GovClaimModeDirect

	_, _, err := newService(&fakeSourceReader{}, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{}).
		RenderZip(context.Background(), uuid.New())

	assert.ErrorIs(t, err, app.ErrNotZipJob)
}

func TestRenderCaseFile_RestoresNationalIDsFromCiphers(t *testing.T) {
	caseA := uuid.New()
	driver := uuid.New()
	store := &fakeExportStore{jobID: uuid.New()}
	store.ciphers = app.NationalIDCiphers{
		Case:    mustEncrypt(t, "A202559750"),
		Drivers: map[uuid.UUID][]byte{driver: mustEncrypt(t, "K120098177")},
	}
	renderer := &recordingRenderer{}
	svc := newService(
		&fakeSourceReader{sources: []app.GovClaimSource{newSource(t, caseA, "C001", "蔡曾切", driver, 1, 1, "outbound", "09:40")}},
		store, renderer, &recordingArchiver{}, stubPrecheckRepo{},
	)

	job, err := svc.CreateGovClaimJob(context.Background(), newInput(app.GovClaimModeDirect, caseA))
	require.NoError(t, err)

	file, err := svc.RenderCaseFile(context.Background(), job.ID, caseA)
	require.NoError(t, err)
	assert.Equal(t, "蔡曾切11507.xlsx", file.FileName)
	assert.NotEmpty(t, file.Bytes)

	require.Len(t, renderer.batches, 2)
	rebuilt := renderer.batches[1]
	require.Len(t, rebuilt, 1)
	assert.Equal(t, "A202559750", rebuilt[0].Cells[0])
	assert.Equal(t, "K120098177", rebuilt[0].Cells[6])
	assert.Equal(t, renderer.batches[0][0].Cells, rebuilt[0].Cells, "由快照重繪的列必須與當初產生的列完全相同")
}

func TestRenderCaseFile_UnknownCaseReturnsNotFound(t *testing.T) {
	store := &fakeExportStore{jobID: uuid.New()}

	_, err := newService(&fakeSourceReader{}, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{}).
		RenderCaseFile(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, app.ErrExportFileNotFound)
}

func TestRenderCaseFile_UsesImmutableFileContent(t *testing.T) {
	jobID := uuid.New()
	caseID := uuid.New()
	renderer := &recordingRenderer{}
	store := &immutableExportStore{
		fakeExportStore: &fakeExportStore{
			jobID: jobID,
			storedJob: app.GovClaimJob{
				ID: jobID,
				Files: []app.GovClaimCaseFile{{
					CaseID:   caseID,
					FileName: "蔡曾切11507.xlsx",
				}},
			},
		},
		content: []byte("immutable-xlsx"),
	}

	file, err := newService(&fakeSourceReader{}, store, renderer, &recordingArchiver{}, stubPrecheckRepo{}).
		RenderCaseFile(context.Background(), jobID, caseID)

	require.NoError(t, err)
	assert.Equal(t, []byte("immutable-xlsx"), file.Bytes)
	assert.Empty(t, renderer.batches, "有 immutable 檔案時不應重新產檔")
}

func TestRenderCaseFile_PropagatesImmutableFileError(t *testing.T) {
	jobID := uuid.New()
	caseID := uuid.New()
	immutableErr := errors.New("immutable file unavailable")
	store := &immutableExportStore{
		fakeExportStore: &fakeExportStore{
			jobID: jobID,
			storedJob: app.GovClaimJob{
				ID: jobID,
				Files: []app.GovClaimCaseFile{{
					CaseID:   caseID,
					FileName: "蔡曾切11507.xlsx",
				}},
			},
		},
		err: immutableErr,
	}

	_, err := newService(&fakeSourceReader{}, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{}).
		RenderCaseFile(context.Background(), jobID, caseID)

	require.Error(t, err)
	assert.ErrorIs(t, err, immutableErr)
	assert.Contains(t, err.Error(), "load immutable export file")
}

func TestRenderCaseFile_ReturnsSnapshotROCDateError(t *testing.T) {
	jobID := uuid.New()
	caseID := uuid.New()
	store := &fakeExportStore{
		jobID: jobID,
		storedJob: app.GovClaimJob{
			ID:    jobID,
			Files: []app.GovClaimCaseFile{{CaseID: caseID, FileName: "蔡曾切11507.xlsx"}},
		},
		caseLines: map[uuid.UUID][]app.ExportLine{
			caseID: {{CaseID: caseID, ServiceDateROC: 1150230}},
		},
	}

	_, err := newService(&fakeSourceReader{}, store, &recordingRenderer{}, &recordingArchiver{}, stubPrecheckRepo{}).
		RenderCaseFile(context.Background(), jobID, caseID)

	require.Error(t, err)
	assert.ErrorIs(t, err, rocdate.ErrInvalidROCVal)
	assert.Contains(t, err.Error(), "convert service date from ROC")
}
