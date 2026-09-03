package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/platform/config"
)

const govClaimJobType = "gov_claim"

var periodYMPattern = regexp.MustCompile(`^\d{5}$`)

// CreateGovClaimInput 代表建立政府申報匯出工作的輸入條件。
type CreateGovClaimInput struct {
	PeriodYM      string
	Region        string
	CaseIDs       []uuid.UUID
	Mode          GovClaimMode
	CreatedBy     uuid.UUID
	CreatedByName string
	ActorRole     string
}

// GovClaimService 產生政府申報工作簿：查詢趟次、組出 33 欄申報列、落地快照並輸出檔案。
type GovClaimService struct {
	cfg      *config.Config
	reader   GovClaimSourceReader
	store    ExportJobStore
	renderer Renderer
	archiver Archiver
	precheck *PrecheckService
	audit    AuditWriter
}

// NewGovClaimService 建立 GovClaimService 實例。
func NewGovClaimService(
	cfg *config.Config,
	reader GovClaimSourceReader,
	store ExportJobStore,
	renderer Renderer,
	archiver Archiver,
	precheck *PrecheckService,
	audit AuditWriter,
) *GovClaimService {
	return &GovClaimService{
		cfg:      cfg,
		reader:   reader,
		store:    store,
		renderer: renderer,
		archiver: archiver,
		precheck: precheck,
		audit:    audit,
	}
}

// CreateGovClaimJob 前置檢核通過後同步產生逐案申報工作簿，並將申報列快照與檔案中繼資料落地。
// 檢核有阻斷性錯誤時回傳 ErrPrecheckBlocked，且不建立任何工作紀錄。
func (s *GovClaimService) CreateGovClaimJob(ctx context.Context, input CreateGovClaimInput) (GovClaimJob, error) {
	periodYM, start, end, err := parsePeriodYM(input.PeriodYM)
	if err != nil {
		return GovClaimJob{}, err
	}

	report, err := s.precheck.RunPrecheck(ctx, periodYM, input.Region)
	if err != nil {
		return GovClaimJob{}, fmt.Errorf("run precheck: %w", err)
	}
	if !report.Passed {
		return GovClaimJob{}, ErrPrecheckBlocked
	}

	format := "xlsx"
	if input.Mode == GovClaimModeZip {
		format = "zip"
	}

	jobID, err := s.store.CreateJob(ctx, ExportJobCreate{
		JobType:       govClaimJobType,
		PeriodYM:      periodYM,
		Region:        input.Region,
		Format:        format,
		CaseIDs:       input.CaseIDs,
		Precheck:      report,
		CreatedBy:     input.CreatedBy,
		CreatedByName: input.CreatedByName,
	})
	if err != nil {
		return GovClaimJob{}, fmt.Errorf("create export job: %w", err)
	}

	files, lines, skipped, err := s.buildJobContent(ctx, periodYM, input, start, end)
	if err != nil {
		// 產檔失敗仍要留下失敗紀錄，讓使用者在歷史清單看得到這次嘗試
		if failErr := s.store.FailJob(ctx, jobID, exportFailureMessage(err)); failErr != nil {
			return GovClaimJob{}, fmt.Errorf("mark export job failed: %w", failErr)
		}
		return GovClaimJob{}, err
	}

	if err := s.store.CompleteJob(ctx, jobID, files, lines); err != nil {
		return GovClaimJob{}, fmt.Errorf("complete export job: %w", err)
	}

	job, err := s.store.GetJob(ctx, jobID)
	if err != nil {
		return GovClaimJob{}, fmt.Errorf("reload export job: %w", err)
	}
	job.Skipped = skipped

	s.recordExportAudit(ctx, job, input)
	return job, nil
}

// recordExportAudit 留下一筆匯出稽核紀錄；稽核寫入失敗不影響已完成的匯出結果，僅盡力而為。
func (s *GovClaimService) recordExportAudit(ctx context.Context, job GovClaimJob, input CreateGovClaimInput) {
	entityID := job.ID.String()
	var actorID *uuid.UUID
	if input.CreatedBy != uuid.Nil {
		actorID = &input.CreatedBy
	}
	var actorRole *string
	if input.ActorRole != "" {
		actorRole = &input.ActorRole
	}

	cases := make([]ExportJobAuditCaseFile, 0, len(job.Files))
	for _, f := range job.Files {
		cases = append(cases, ExportJobAuditCaseFile{
			CaseName: f.CaseName,
			Region:   f.Region,
			FileName: f.FileName,
			RowCount: f.RowCount,
		})
	}

	_ = s.audit.Write(ctx, AuditEntry{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     "export",
		EntityType: "export_jobs",
		EntityID:   &entityID,
		AfterData: ExportJobAuditSnapshot{
			PeriodYM:   job.PeriodYM,
			Region:     job.Region,
			Mode:       string(job.Mode),
			TotalCases: job.TotalCases,
			TotalRows:  job.TotalRows,
			Cases:      cases,
		},
	})
}

// GetGovClaimJob 取得單筆匯出工作與其逐案檔案清單。
func (s *GovClaimService) GetGovClaimJob(ctx context.Context, jobID uuid.UUID) (GovClaimJob, error) {
	return s.store.GetJob(ctx, jobID)
}

// ListExportJobs 依建立時間新到舊列出匯出工作歷史；分頁參數由 transport 夾限後傳入。
func (s *GovClaimService) ListExportJobs(ctx context.Context, page, pageSize int) ([]GovClaimJob, int64, error) {
	return s.store.ListJobs(ctx, page, pageSize)
}

// RenderCaseFile 由申報列快照重繪指定個案的工作簿位元組。
func (s *GovClaimService) RenderCaseFile(ctx context.Context, jobID, caseID uuid.UUID) (GovClaimCaseFile, error) {
	job, err := s.store.GetJob(ctx, jobID)
	if err != nil {
		return GovClaimCaseFile{}, err
	}

	target, found := findCaseFile(job.Files, caseID)
	if !found {
		return GovClaimCaseFile{}, ErrExportFileNotFound
	}

	content, err := s.renderSnapshot(ctx, jobID, caseID)
	if err != nil {
		return GovClaimCaseFile{}, err
	}
	target.Bytes = content
	return target, nil
}

// RenderZip 把該工作的所有個案工作簿打包成單一壓縮檔；非壓縮檔模式回傳 ErrNotZipJob。
func (s *GovClaimService) RenderZip(ctx context.Context, jobID uuid.UUID) (string, []byte, error) {
	job, err := s.store.GetJob(ctx, jobID)
	if err != nil {
		return "", nil, err
	}
	if job.Mode != GovClaimModeZip {
		return "", nil, ErrNotZipJob
	}

	entries := make([]ZipEntry, 0, len(job.Files))
	for _, f := range job.Files {
		content, err := s.renderSnapshot(ctx, jobID, f.CaseID)
		if err != nil {
			return "", nil, err
		}
		entries = append(entries, ZipEntry{Name: f.FileName, Content: content})
	}

	archive, err := s.archiver.BuildZip(entries)
	if err != nil {
		return "", nil, fmt.Errorf("build zip: %w", err)
	}
	return ZipFileName(job.Region, job.PeriodYM), archive, nil
}

// ZipFileName 組出壓縮檔檔名；未指定地區時以 all 標示。
func ZipFileName(region, periodYM string) string {
	if region == "" {
		region = "all"
	}
	return fmt.Sprintf("gov-claim-%s-%s.zip", region, periodYM)
}

// buildJobContent 查詢趟次並組出逐案工作簿、申報列快照與跳過清單。
func (s *GovClaimService) buildJobContent(
	ctx context.Context,
	periodYM string,
	input CreateGovClaimInput,
	start, end time.Time,
) ([]GovClaimCaseFile, []ExportLine, []ClaimSkip, error) {
	sources, err := s.reader.QueryGovClaimSources(ctx, start, end, input.Region, input.CaseIDs)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query gov claim sources: %w", err)
	}

	groups := groupByCase(sources)
	skips := newSkipTally()

	files := make([]GovClaimCaseFile, 0, len(groups))
	lines := make([]ExportLine, 0, len(sources))
	usedFileNames := make(map[string]bool)
	lineNo := 0

	for _, group := range groups {
		rows, rowDrivers := s.buildCaseRows(group, skips)
		if len(rows) == 0 {
			continue
		}
		govform.SortClaimRows(rows, false)

		content, err := s.renderer.RenderGovClaim(rows)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("render gov claim for case %s: %w", group.caseName, err)
		}

		sum := sha256.Sum256(content)
		files = append(files, GovClaimCaseFile{
			CaseID:   group.caseID,
			CaseName: group.caseName,
			Region:   group.region,
			FileName: uniqueFileName(usedFileNames, group.caseName, periodYM),
			RowCount: len(rows),
			Checksum: hex.EncodeToString(sum[:]),
			Bytes:    content,
		})

		for _, row := range rows {
			lineNo++
			lines = append(lines, newExportLine(lineNo, group, row, rowDrivers[rowKeyOf(row)]))
		}
	}

	if len(files) == 0 {
		return nil, nil, nil, ErrNoClaimRows
	}
	return files, lines, skips.list(), nil
}

// buildCaseRows 逐筆驗證來源資料並轉成 33 欄申報列。
// 缺欄位的趟次計入跳過清單，不讓 govform.BuildClaimRow 的相容預設值把它補成一列看似正常的申報資料。
func (s *GovClaimService) buildCaseRows(group caseGroup, skips *skipTally) ([]govform.ClaimRow, map[rowKey]*uuid.UUID) {
	rowDrivers := make(map[rowKey]*uuid.UUID, len(group.items))

	caseNationalID, err := s.decrypt(group.nationalIDCipher)
	if err != nil || caseNationalID == "" {
		skips.add(group, SkipReasonNoNationalID, len(group.items))
		return nil, rowDrivers
	}

	driverIDCache := make(map[uuid.UUID]string)
	rows := make([]govform.ClaimRow, 0, len(group.items))

	for _, item := range group.items {
		reason, ok := validateSource(item)
		if !ok {
			skips.add(group, reason, 1)
			continue
		}

		driverNationalID, err := s.driverNationalID(driverIDCache, *item.DriverID, item.DriverNationalIDCipher)
		if err != nil || driverNationalID == "" {
			skips.add(group, SkipReasonNoDriver, 1)
			continue
		}

		departAt, err := combineDepartTime(item.ServiceDate, *item.DepartTime)
		if err != nil {
			skips.add(group, SkipReasonNoDepartTime, 1)
			continue
		}

		row, err := govform.BuildClaimRow(govform.ClaimRowInput{
			NationalIDPlain:  caseNationalID,
			ServiceDate:      item.ServiceDate,
			ServiceCode:      item.ServiceCode,
			ServiceCategory:  *item.ServiceCategory,
			UnitPrice:        item.UnitPrice,
			DriverNationalID: driverNationalID,
			DepartTime:       departAt,
			DurationMin:      *item.DurationMin,
			NotClaimedAA09:   item.NotClaimedAA09,
			Direction:        *item.Direction,
			LegSeq:           item.LegSeq,
			HomeAddress:      item.HomeAddress,
			SiteAddress:      item.SiteAddress,
			DistanceKM:       item.DistanceKM,
			PlateNo:          item.PlateNo,
			ServiceUsageType: *item.ServiceUsageType,
		})
		if err != nil {
			skips.add(group, SkipReasonBuildRowFailed, 1)
			continue
		}

		rows = append(rows, row)
		rowDrivers[rowKeyOf(row)] = item.DriverID
	}

	return rows, rowDrivers
}

// renderSnapshot 讀回快照、補回兩個身分證欄位後重繪工作簿。
func (s *GovClaimService) renderSnapshot(ctx context.Context, jobID, caseID uuid.UUID) ([]byte, error) {
	lines, err := s.store.LoadCaseLines(ctx, jobID, caseID)
	if err != nil {
		return nil, fmt.Errorf("load export lines: %w", err)
	}
	if len(lines) == 0 {
		return nil, ErrExportFileNotFound
	}

	ciphers, err := s.store.LoadNationalIDCiphers(ctx, caseID, collectDriverIDs(lines))
	if err != nil {
		return nil, fmt.Errorf("load national id ciphers: %w", err)
	}
	caseNationalID, err := s.decrypt(ciphers.Case)
	if err != nil {
		return nil, fmt.Errorf("decrypt case national id: %w", err)
	}

	rows := make([]govform.ClaimRow, 0, len(lines))
	for _, line := range lines {
		row := govform.ClaimRow{
			Cells:      line.Payload.Cells,
			Direction:  line.Payload.Direction,
			LegSeq:     line.Payload.LegSeq,
			NationalID: caseNationalID,
		}
		row.Cells[0] = caseNationalID
		row.Cells[6] = ""
		if line.Payload.DriverID != nil {
			driverNationalID, err := s.decrypt(ciphers.Drivers[*line.Payload.DriverID])
			if err != nil {
				return nil, fmt.Errorf("decrypt driver national id: %w", err)
			}
			row.Cells[6] = driverNationalID
		}
		if serviceDate, err := rocdate.FromROC(line.ServiceDateROC); err == nil {
			row.ServiceDate = serviceDate
		}
		rows = append(rows, row)
	}

	content, err := s.renderer.RenderGovClaim(rows)
	if err != nil {
		return nil, fmt.Errorf("render gov claim snapshot: %w", err)
	}
	return content, nil
}

func (s *GovClaimService) decrypt(cipher []byte) (string, error) {
	if len(cipher) == 0 {
		return "", nil
	}
	return crypto.Decrypt(cipher, s.cfg.EncryptionKey)
}

// driverNationalID 讓同一位司機在整個個案內只解密一次。
func (s *GovClaimService) driverNationalID(cache map[uuid.UUID]string, driverID uuid.UUID, cipher []byte) (string, error) {
	if plain, ok := cache[driverID]; ok {
		return plain, nil
	}
	plain, err := s.decrypt(cipher)
	if err != nil {
		return "", err
	}
	cache[driverID] = plain
	return plain, nil
}

// validateSource 檢查該趟次是否具備組出合法申報列的最小資料集。
func validateSource(item GovClaimSource) (string, bool) {
	switch {
	case item.Direction == nil || *item.Direction == "":
		return SkipReasonNoScheduleLeg, false
	case item.DepartTime == nil || *item.DepartTime == "":
		return SkipReasonNoDepartTime, false
	case item.DurationMin == nil || *item.DurationMin <= 0:
		return SkipReasonNoDepartTime, false
	case item.DriverID == nil || len(item.DriverNationalIDCipher) == 0:
		return SkipReasonNoDriver, false
	case item.ServiceCategory == nil || (*item.ServiceCategory != 1 && *item.ServiceCategory != 2):
		return SkipReasonNoServiceCategory, false
	case item.ServiceUsageType == nil || *item.ServiceUsageType < 1 || *item.ServiceUsageType > 4:
		return SkipReasonNoUsageType, false
	case item.UnitPrice <= 0:
		return SkipReasonNoUnitPrice, false
	}
	return "", true
}

// parsePeriodYM 將民國年月正規化為 5 碼並換算成該月的西元起訖日（左閉右開）。
func parsePeriodYM(raw string) (string, time.Time, time.Time, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), "-", "")
	if !periodYMPattern.MatchString(normalized) {
		return "", time.Time{}, time.Time{}, ErrInvalidPeriodYM
	}

	rocYear := parseDigits(normalized[:3])
	month := parseDigits(normalized[3:])
	if rocYear < 1 || month < 1 || month > 12 {
		return "", time.Time{}, time.Time{}, ErrInvalidPeriodYM
	}

	start := time.Date(rocYear+1911, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return normalized, start, start.AddDate(0, 1, 0), nil
}

func combineDepartTime(serviceDate time.Time, hhmm string) (time.Time, error) {
	parsed, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse depart time %q: %w", hhmm, err)
	}
	return time.Date(
		serviceDate.Year(), serviceDate.Month(), serviceDate.Day(),
		parsed.Hour(), parsed.Minute(), 0, 0, time.UTC,
	), nil
}

// newExportLine 組出落地用的申報列快照；兩個身分證欄位一律清空，改以 driverId 保留關聯。
func newExportLine(lineNo int, group caseGroup, row govform.ClaimRow, driverID *uuid.UUID) ExportLine {
	payload := ClaimLinePayload{
		Cells:     row.Cells,
		DriverID:  driverID,
		Direction: row.Direction,
		LegSeq:    row.LegSeq,
	}
	payload.Cells[0] = ""
	payload.Cells[6] = ""

	serviceDateROC, _ := rocdate.ToROC(row.ServiceDate)
	return ExportLine{
		LineNo:           lineNo,
		CaseID:           group.caseID,
		NationalIDMasked: group.nationalIDMasked,
		ServiceDateROC:   serviceDateROC,
		Payload:          payload,
	}
}

// uniqueFileName 以「個案姓名＋民國年月」命名，比照政府端收到的範本檔名；同名時補序號。
func uniqueFileName(used map[string]bool, caseName, periodYM string) string {
	base := fmt.Sprintf("%s%s", caseName, periodYM)
	name := base + ".xlsx"
	counter := 2
	for used[name] {
		name = fmt.Sprintf("%s (%d).xlsx", base, counter)
		counter++
	}
	used[name] = true
	return name
}

func exportFailureMessage(err error) string {
	if errors.Is(err, ErrNoClaimRows) {
		return "指定條件下沒有可申報的搭乘紀錄"
	}
	return "產生申報檔案失敗"
}

func findCaseFile(files []GovClaimCaseFile, caseID uuid.UUID) (GovClaimCaseFile, bool) {
	for _, f := range files {
		if f.CaseID == caseID {
			return f, true
		}
	}
	return GovClaimCaseFile{}, false
}

func collectDriverIDs(lines []ExportLine) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(lines))
	ids := make([]uuid.UUID, 0, len(lines))
	for _, line := range lines {
		if line.Payload.DriverID == nil || seen[*line.Payload.DriverID] {
			continue
		}
		seen[*line.Payload.DriverID] = true
		ids = append(ids, *line.Payload.DriverID)
	}
	return ids
}

func parseDigits(s string) int {
	result := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		result = result*10 + int(ch-'0')
	}
	return result
}

type rowKey struct {
	serviceDate string
	legSeq      int16
}

func rowKeyOf(row govform.ClaimRow) rowKey {
	return rowKey{serviceDate: row.ServiceDate.Format("2006-01-02"), legSeq: row.LegSeq}
}

type caseGroup struct {
	caseID           uuid.UUID
	caseName         string
	region           string
	nationalIDCipher []byte
	nationalIDMasked string
	items            []GovClaimSource
}

// groupByCase 依個案分組並以個案姓名與 ID 排序，讓同一組輸入永遠產出相同的檔案順序。
func groupByCase(sources []GovClaimSource) []caseGroup {
	index := make(map[uuid.UUID]int)
	groups := make([]caseGroup, 0)

	for _, item := range sources {
		pos, ok := index[item.CaseID]
		if !ok {
			groups = append(groups, caseGroup{
				caseID:           item.CaseID,
				caseName:         item.CaseName,
				region:           item.Region,
				nationalIDCipher: item.CaseNationalIDCipher,
				nationalIDMasked: item.CaseNationalIDMasked,
			})
			pos = len(groups) - 1
			index[item.CaseID] = pos
		}
		groups[pos].items = append(groups[pos].items, item)
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].caseName != groups[j].caseName {
			return groups[i].caseName < groups[j].caseName
		}
		return groups[i].caseID.String() < groups[j].caseID.String()
	})
	return groups
}

type skipTally struct {
	order []string
	byKey map[string]*ClaimSkip
}

func newSkipTally() *skipTally {
	return &skipTally{byKey: make(map[string]*ClaimSkip)}
}

func (t *skipTally) add(group caseGroup, reason string, count int) {
	key := group.caseID.String() + "|" + reason
	if existing, ok := t.byKey[key]; ok {
		existing.Count += count
		return
	}
	t.byKey[key] = &ClaimSkip{CaseID: group.caseID, CaseName: group.caseName, Reason: reason, Count: count}
	t.order = append(t.order, key)
}

func (t *skipTally) list() []ClaimSkip {
	result := make([]ClaimSkip, 0, len(t.order))
	for _, key := range t.order {
		result = append(result, *t.byKey[key])
	}
	return result
}
