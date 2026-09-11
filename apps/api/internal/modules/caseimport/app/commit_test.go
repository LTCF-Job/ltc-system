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
	svc := &ImportService{cases: registrar, duplicateStager: stager, txRunner: fakeTxRunner{}}

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
	svc := &ImportService{cases: registrar, duplicateStager: stager, txRunner: fakeTxRunner{}}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "重複重試列", IsDuplicate: true, DuplicateCaseID: &dupID},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 0, result.StagedDuplicateCount)
	assert.Equal(t, 1, result.AlreadyImportedCount)
	assert.Empty(t, stager.staged)
}

func TestCommitCases_CreatesCaseWhenSiteAndCaregiverNamesDoNotMatch(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	svc := &ImportService{
		cases:         registrar,
		siteRepo:      fakeSiteLookup{byName: map[string]uuid.UUID{}},
		caregiverRepo: fakeCaregiverLookup{},
		txRunner:      fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案甲", SiteName: "查無此單位", CareContactRole: "個管", CareContactName: "查無此人"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount, "據點／照護人員比對不到仍應建立個案")
	assert.Empty(t, result.SkippedRows)
	require.Len(t, result.Warnings, 2, "據點與照護人員各自獨立比對不到，各附一則警示")

	require.Len(t, registrar.created, 1)
	created := registrar.created[0]
	assert.Nil(t, created.SiteID)
	assert.Equal(t, "查無此單位", created.SiteNameRaw)
	// 比對不到時保留工作表原始角色文字，caregiver_pending 才有線索可供待維護頁顯示。
	assert.Nil(t, created.CaregiverID)
	require.NotNil(t, created.CareContactRole)
	assert.Equal(t, "個管", *created.CareContactRole)
}

// 同名唯一命中：角色一律以主檔為準，即使工作表填的是另一個角色也不採用。
func TestCommitCases_CaregiverMatchedByNameOverridesSheetRole(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	caregiverID := uuid.New()
	svc := &ImportService{
		cases:    registrar,
		siteRepo: fakeSiteLookup{byName: map[string]uuid.UUID{}},
		caregiverRepo: fakeCaregiverLookup{byName: map[string][]CaregiverRef{
			"陳小華": {{ID: caregiverID, Name: "陳小華", Type: "specialist"}},
		}},
		txRunner: fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案甲", SiteName: "查無此單位", CareContactRole: "個管", CareContactName: "陳小華"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	require.Len(t, registrar.created, 1)
	created := registrar.created[0]
	require.NotNil(t, created.CaregiverID)
	assert.Equal(t, caregiverID, *created.CaregiverID)
	require.NotNil(t, created.CareContactRole)
	assert.Equal(t, "照專", *created.CareContactRole)
}

// 同名多筆時才用工作表的「個管or照專」消歧；消歧成功即採用該筆。
func TestCommitCases_CaregiverDuplicateNamesResolvedByRole(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	managerID, specialistID := uuid.New(), uuid.New()
	svc := &ImportService{
		cases:    registrar,
		siteRepo: fakeSiteLookup{byName: map[string]uuid.UUID{}},
		caregiverRepo: fakeCaregiverLookup{byName: map[string][]CaregiverRef{
			"陳小華": {
				{ID: managerID, Name: "陳小華", Type: "case_manager"},
				{ID: specialistID, Name: "陳小華", Type: "specialist"},
			},
		}},
		txRunner: fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案甲", SiteName: "查無此單位", CareContactRole: "照專", CareContactName: "陳小華"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	require.Len(t, registrar.created, 1)
	require.NotNil(t, registrar.created[0].CaregiverID)
	assert.Equal(t, specialistID, *registrar.created[0].CaregiverID)
}

// 同名多筆但角色欄空白：無法唯一對應，落入待維護而不是隨便挑一筆。
func TestCommitCases_CaregiverDuplicateNamesWithoutRoleStaysPending(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	svc := &ImportService{
		cases:    registrar,
		siteRepo: fakeSiteLookup{byName: map[string]uuid.UUID{}},
		caregiverRepo: fakeCaregiverLookup{byName: map[string][]CaregiverRef{
			"陳小華": {
				{ID: uuid.New(), Name: "陳小華", Type: "case_manager"},
				{ID: uuid.New(), Name: "陳小華", Type: "specialist"},
			},
		}},
		txRunner: fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案甲", SiteName: "查無此單位", CareContactName: "陳小華"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	require.Len(t, registrar.created, 1)
	assert.Nil(t, registrar.created[0].CaregiverID)
	require.Len(t, result.Warnings, 2)
}

// 照護人員姓名空白：不是錯誤也不是待維護，不得產生警示。
func TestCommitCases_BlankCaregiverNameProducesNoWarning(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	siteID := uuid.New()
	svc := &ImportService{
		cases:         registrar,
		siteRepo:      fakeSiteLookup{byName: map[string]uuid.UUID{"竹南日照單位": siteID}},
		caregiverRepo: fakeCaregiverLookup{},
		txRunner:      fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案甲", SiteName: "竹南日照單位"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Empty(t, result.Warnings)
	require.Len(t, registrar.created, 1)
	assert.Nil(t, registrar.created[0].CaregiverID)
}

// 據點欄位完全空白（使用者根本沒填，不是「填了但比對不到」）時，SiteID 與 SiteNameRaw
// 不可同時為空，否則違反 cases 的 ck_cases_site_present CHECK 約束；必須落入哨兵值待維護，
// 而不是讓建立個案的 INSERT 直接失敗。
func TestCommitCases_BlankSiteNameFallsBackToPendingPlaceholder(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	svc := &ImportService{
		cases:    registrar,
		siteRepo: fakeSiteLookup{byName: map[string]uuid.UUID{}},
		txRunner: fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案空據點", SiteName: ""},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount, "據點空白仍應建立個案，落入待維護而非匯入失敗")
	assert.Empty(t, result.SkippedRows)
	assert.Empty(t, result.FailedRows)

	require.Len(t, registrar.created, 1)
	created := registrar.created[0]
	assert.Nil(t, created.SiteID)
	assert.NotEmpty(t, created.SiteNameRaw, "SiteID 與 SiteNameRaw 不可同時為空，否則違反 ck_cases_site_present")
}

// 疑似重複個案裁決為新個案時，同樣要套用哨兵值，否則 resolveDuplicateAsNewCase 的
// INSERT 會因兩欄同時為空而違反 CHECK 約束。
func TestCommitCases_BlankSiteNameOnDuplicateRowAlsoGetsPlaceholder(t *testing.T) {
	stager := &fakeDuplicateCandidateStager{}
	dupID := uuid.New()
	svc := &ImportService{
		cases:           &fakeCaseRegistrar{},
		siteRepo:        fakeSiteLookup{byName: map[string]uuid.UUID{}},
		duplicateStager: stager,
		txRunner:        fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案空據點重複", SiteName: "", IsDuplicate: true, DuplicateCaseID: &dupID},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.StagedDuplicateCount)
	require.Len(t, stager.staged, 1)
	assert.Nil(t, stager.staged[0].SiteID)
	assert.NotEmpty(t, stager.staged[0].SiteNameRaw, "暫存列的 SiteID 與 SiteNameRaw 不可同時為空")
}

func TestCommitCases_ResolvesSiteAndCaregiverWhenNamesMatch(t *testing.T) {
	registrar := &fakeCaseRegistrar{}
	siteID := uuid.New()
	caregiverID := uuid.New()
	svc := &ImportService{
		cases:    registrar,
		siteRepo: fakeSiteLookup{byName: map[string]uuid.UUID{"竹南日照單位": siteID}},
		caregiverRepo: fakeCaregiverLookup{byName: map[string][]CaregiverRef{
			"陳小華": {{ID: caregiverID, Name: "陳小華", Type: "case_manager"}},
		}},
		txRunner: fakeTxRunner{},
	}

	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "個案乙", SiteName: "竹南日照單位", CareContactRole: "個管", CareContactName: "陳小華", InboundVehicle: "查無此車回"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Empty(t, result.Warnings, "據點與照護人員都比對到，接送車輛不參與比對")

	require.Len(t, registrar.created, 1)
	created := registrar.created[0]
	require.NotNil(t, created.SiteID)
	assert.Equal(t, siteID, *created.SiteID)
	require.NotNil(t, created.CaregiverID)
	assert.Equal(t, caregiverID, *created.CaregiverID)
}

// fakeCaregiverLookup 以姓名回傳同名清單；未登錄的姓名回傳空清單代表查無此人。
type fakeCaregiverLookup struct{ byName map[string][]CaregiverRef }

func (f fakeCaregiverLookup) FindByName(ctx context.Context, name string) ([]CaregiverRef, error) {
	return f.byName[name], nil
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

// duplicateNationalIDCaseRegistrar 模擬 casemgmt 在唯一鍵衝突時回傳的底層錯誤
// （案文字與 casemgmt/app.ErrDuplicateNationalID 一致，見 commit.go 的
// caseRegistrarDuplicateNationalIDText 說明），驗證 commit.go 能把它改寫成
// 「身分證字號與本檔其他列重複」而非通用失敗訊息。
type duplicateNationalIDCaseRegistrar struct{}

func (duplicateNationalIDCaseRegistrar) CreateCase(ctx context.Context, in NewCase, actor Actor) (uuid.UUID, error) {
	return uuid.Nil, errors.New("national id already exists")
}

func (duplicateNationalIDCaseRegistrar) RecordSkipped(ctx context.Context, row CaseImportSkippedRow, actor Actor) {
}

func TestCommitCases_DuplicateNationalIDWithinFileGetsSpecificReason(t *testing.T) {
	svc := &ImportService{
		cases:    duplicateNationalIDCaseRegistrar{},
		txRunner: fakeTxRunner{},
	}
	preview := &CaseImportPreviewResult{Rows: []CaseImportRowResult{
		{RowIndex: 1, Name: "身分證字號重複列"},
	}}

	result, err := svc.CommitCases(context.Background(), preview, Actor{ActorID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, result.FailedCount)
	require.Len(t, result.FailedRows, 1)
	require.Len(t, result.FailedRows[0].Reasons, 1)
	assert.Equal(t, "身分證字號與本檔其他列重複", result.FailedRows[0].Reasons[0])
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
	svc := &ImportService{cases: registrar, txRunner: fakeTxRunner{}}

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
