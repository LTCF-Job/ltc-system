package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
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
	CaseIDs       []uuid.UUID
	Mode          GovClaimMode
	CreatedBy     uuid.UUID
	CreatedByName string
	ActorRole     string
}

// GovClaimService 產生政府申報工作簿：查詢趟次、組出 33 欄申報列、寫入快照並輸出檔案。
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

// CreateGovClaimJob 前置檢核通過後同步產生逐案申報工作簿，並將申報列快照與檔案中繼資料寫入。
// 檢核有阻斷性錯誤時回傳 ErrPrecheckBlocked，且不建立任何工作紀錄。
func (s *GovClaimService) CreateGovClaimJob(ctx context.Context, input CreateGovClaimInput) (GovClaimJob, error) {
	if input.Mode != GovClaimModeDirect && input.Mode != GovClaimModeZip {
		return GovClaimJob{}, ErrInvalidExportMode
	}
	if len(input.CaseIDs) == 0 {
		return GovClaimJob{}, ErrCaseIDsRequired
	}
	periodYM, start, end, err := parsePeriodYM(input.PeriodYM)
	if err != nil {
		return GovClaimJob{}, err
	}

	scope := NewClaimScope(start, end, input.CaseIDs)
	report, err := s.precheck.RunPrecheck(ctx, scope)
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
		Format:        format,
		CaseIDs:       input.CaseIDs,
		Precheck:      report,
		CreatedBy:     input.CreatedBy,
		CreatedByName: input.CreatedByName,
	})
	if err != nil {
		return GovClaimJob{}, fmt.Errorf("create export job: %w", err)
	}
	s.recordExportAuditBestEffort(ctx, GovClaimJob{
		ID:       jobID,
		PeriodYM: periodYM,
		Mode:     input.Mode,
		Status:   ExportStatusRunning,
	}, input, "export_requested")

	files, lines, dataGaps, err := s.buildJobContent(ctx, periodYM, input, scope)
	if err != nil {
		// 產檔失敗仍要留下失敗紀錄，讓使用者在歷史清單看得到這次嘗試
		if failErr := s.store.FailJob(ctx, jobID, exportFailureMessage(err)); failErr != nil {
			return GovClaimJob{}, fmt.Errorf("mark export job failed: %w", failErr)
		}
		s.recordExportAuditBestEffort(ctx, GovClaimJob{
			ID:           jobID,
			PeriodYM:     periodYM,
			Mode:         input.Mode,
			Status:       ExportStatusFailed,
			ErrorMessage: exportFailureMessage(err),
		}, input, "export_failed")
		return GovClaimJob{}, err
	}

	if err := s.store.CompleteJob(ctx, jobID, files, lines); err != nil {
		if failErr := s.store.FailJob(ctx, jobID, exportFailureMessage(err)); failErr != nil {
			return GovClaimJob{}, fmt.Errorf("mark export job failed: %w", failErr)
		}
		s.recordExportAuditBestEffort(ctx, GovClaimJob{
			ID:           jobID,
			PeriodYM:     periodYM,
			Mode:         input.Mode,
			Status:       ExportStatusFailed,
			ErrorMessage: exportFailureMessage(err),
		}, input, "export_failed")
		return GovClaimJob{}, fmt.Errorf("complete export job: %w", err)
	}

	// 成功稽核必須在 CompleteJob 成功後寫入；稽核系統短暫故障不得把已完成的
	// 匯出改報成失敗，背景 log 會保留補寫線索。
	s.recordExportAuditBestEffort(ctx, GovClaimJob{
		ID:         jobID,
		PeriodYM:   periodYM,
		Mode:       input.Mode,
		Status:     ExportStatusSucceeded,
		TotalCases: len(files),
		TotalRows:  len(lines),
		Files:      files,
	}, input, "export_succeeded")

	job, err := s.store.GetJob(ctx, jobID)
	if err != nil {
		return GovClaimJob{}, fmt.Errorf("reload export job: %w", err)
	}
	job.DataGaps = dataGaps

	return job, nil
}

// recordExportAudit 留下一筆政府申報匯出稽核紀錄。
func (s *GovClaimService) recordExportAudit(ctx context.Context, job GovClaimJob, input CreateGovClaimInput, action string) error {
	if s.audit == nil {
		return errors.New("export audit is unavailable")
	}
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
			FileName: f.FileName,
			RowCount: f.RowCount,
		})
	}

	return s.audit.Write(ctx, AuditEntry{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     action,
		EntityType: "export_jobs",
		EntityID:   &entityID,
		AfterData: ExportJobAuditSnapshot{
			Status:     job.Status,
			PeriodYM:   job.PeriodYM,
			Mode:       string(job.Mode),
			TotalCases: job.TotalCases,
			TotalRows:  job.TotalRows,
			Cases:      cases,
		},
	})
}

func (s *GovClaimService) recordExportAuditBestEffort(ctx context.Context, job GovClaimJob, input CreateGovClaimInput, action string) {
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := s.recordExportAudit(auditCtx, job, input, action); err != nil {
		slog.Warn("export_audit_write_failed", slog.String("action", action), slog.String("job_id", job.ID.String()), slog.Any("error", err))
	}
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

	content, err := s.loadImmutableFile(ctx, jobID, caseID)
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
		content, err := s.loadImmutableFile(ctx, jobID, f.CaseID)
		if err != nil {
			return "", nil, err
		}
		entries = append(entries, ZipEntry{Name: f.FileName, Content: content})
	}

	archive, err := s.archiver.BuildZip(entries)
	if err != nil {
		return "", nil, fmt.Errorf("build zip: %w", err)
	}
	return ZipFileName(job.PeriodYM), archive, nil
}

// ZipFileName 組出壓縮檔檔名；未指定地區時以 all 標示。
func ZipFileName(periodYM string) string {
	return fmt.Sprintf("gov-claim-%s.zip", periodYM)
}

// buildJobContent 查詢趟次並組出逐案工作簿、申報列快照與跳過清單。
func (s *GovClaimService) buildJobContent(
	ctx context.Context,
	periodYM string,
	input CreateGovClaimInput,
	scope ClaimScope,
) ([]GovClaimCaseFile, []ExportLine, []ClaimDataGap, error) {
	sources, err := s.reader.QueryGovClaimSources(ctx, scope)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query gov claim sources: %w", err)
	}

	groups := groupByCase(sources)
	gaps := newDataGapTally()

	files := make([]GovClaimCaseFile, 0, len(groups))
	lines := make([]ExportLine, 0, len(sources))
	usedFileNames := make(map[string]bool)
	lineNo := 0

	for _, group := range groups {
		rows, rowDrivers := s.buildCaseRows(group, gaps)
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
			FileName: uniqueFileName(usedFileNames, group.caseName, periodYM),
			RowCount: len(rows),
			Checksum: hex.EncodeToString(sum[:]),
			Bytes:    content,
		})

		for _, row := range rows {
			lineNo++
			line, err := newExportLine(lineNo, group, row, rowDrivers[rowKeyOf(row)])
			if err != nil {
				return nil, nil, nil, fmt.Errorf("build export line for case %s: %w", group.caseName, err)
			}
			lines = append(lines, line)
		}
	}
	// 缺資料的欄位留白後照樣產檔並回報缺漏，不阻擋範圍內其餘資料（含一列都組不出來、
	// 檔案數為 0 的情況）——申報作業本就逐月執行，使用者需要的是先拿到報得出來的資料，
	// 而不是被整批擋下後才回頭排查。
	return files, lines, gaps.list(), nil
}

// buildCaseRows 逐筆把來源資料轉成 33 欄申報列。
// 缺漏的欄位留白並計入資料缺漏清單，該列仍然產出，讓報得出來的資料先進得了申報檔。
func (s *GovClaimService) buildCaseRows(group caseGroup, gaps *dataGapTally) ([]govform.ClaimRow, map[rowKey]*uuid.UUID) {
	rowDrivers := make(map[rowKey]*uuid.UUID, len(group.items))

	caseNationalID, err := s.decrypt(group.nationalIDCipher)
	if err != nil || caseNationalID == "" {
		gaps.add(group, GapReasonNoNationalID, len(group.items))
		caseNationalID = ""
	}

	driverIDCache := make(map[uuid.UUID]string)
	rows := make([]govform.ClaimRow, 0, len(group.items))

	for _, item := range group.items {
		for _, reason := range missingFields(item) {
			gaps.add(group, reason, 1)
		}

		driverNationalID := ""
		if item.DriverID != nil {
			plain, err := s.driverNationalID(driverIDCache, *item.DriverID, item.DriverNationalIDCipher)
			if err != nil || plain == "" {
				gaps.add(group, GapReasonNoDriver, 1)
			} else {
				driverNationalID = plain
			}
		}

		var departAt time.Time
		if item.DepartTime != nil && *item.DepartTime != "" {
			parsed, err := combineDepartTime(item.ServiceDate, *item.DepartTime)
			if err != nil {
				gaps.add(group, GapReasonNoDepartTime, 1)
			} else {
				departAt = parsed
			}
		}

		row, err := govform.BuildClaimRow(govform.ClaimRowInput{
			NationalIDPlain:  caseNationalID,
			ServiceDate:      item.ServiceDate,
			ServiceCode:      item.ServiceCode,
			ServiceCategory:  intOrZero(item.ServiceCategory),
			UnitPrice:        item.UnitPrice,
			DriverNationalID: driverNationalID,
			DepartTime:       departAt,
			DurationMin:      intOrZero(item.DurationMin),
			NotClaimedAA09:   item.NotClaimedAA09,
			Direction:        stringOrEmpty(item.Direction),
			LegSeq:           item.LegSeq,
			HomeAddress:      item.HomeAddress,
			SiteAddress:      item.SiteAddress,
			DistanceKM:       item.DistanceKM,
			PlateNo:          item.PlateNo,
			ServiceUsageType: intOrZero(item.ServiceUsageType),
		})
		if err != nil {
			// 只有連留白都組不出列（如服務日期無法換算民國年）才會少掉這一列。
			gaps.add(group, GapReasonBuildRowFailed, 1)
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
		serviceDate, err := rocdate.FromROC(line.ServiceDateROC)
		if err != nil {
			return nil, fmt.Errorf("convert service date from ROC %d: %w", line.ServiceDateROC, err)
		}
		row.ServiceDate = serviceDate
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

// missingFields 列出該趟次缺漏、將在申報檔留白的欄位；順序即為回報順序。
func missingFields(item GovClaimSource) []string {
	var reasons []string
	if item.Direction == nil || *item.Direction == "" {
		reasons = append(reasons, GapReasonNoScheduleLeg)
	}
	if item.DepartTime == nil || *item.DepartTime == "" || item.DurationMin == nil || *item.DurationMin <= 0 {
		reasons = append(reasons, GapReasonNoDepartTime)
	}
	if item.DriverID == nil || len(item.DriverNationalIDCipher) == 0 {
		reasons = append(reasons, GapReasonNoDriver)
	}
	if item.ServiceCategory == nil || (*item.ServiceCategory != 1 && *item.ServiceCategory != 2) {
		reasons = append(reasons, GapReasonNoServiceCategory)
	}
	if item.ServiceUsageType == nil || *item.ServiceUsageType < 1 || *item.ServiceUsageType > 4 {
		reasons = append(reasons, GapReasonNoUsageType)
	}
	if item.UnitPrice <= 0 {
		reasons = append(reasons, GapReasonNoUnitPrice)
	}
	if strings.TrimSpace(item.ServiceCode) == "" {
		reasons = append(reasons, GapReasonNoServiceCode)
	}
	if strings.TrimSpace(item.HomeAddress) == "" || strings.TrimSpace(item.SiteAddress) == "" {
		reasons = append(reasons, GapReasonNoAddress)
	}
	if item.DistanceKM <= 0 {
		reasons = append(reasons, GapReasonNoDistance)
	}
	if strings.TrimSpace(item.PlateNo) == "" {
		reasons = append(reasons, GapReasonNoPlateNo)
	}
	return reasons
}

func intOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func stringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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

// ParseClaimPeriod 將申報月份轉成標準化月份與日期範圍，供 transport 建立相同 ClaimScope。
func ParseClaimPeriod(raw string) (string, time.Time, time.Time, error) {
	return parsePeriodYM(raw)
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

// newExportLine 組出寫入用的申報列快照；兩個身分證欄位一律清空，改以 driverId 保留關聯。
func newExportLine(lineNo int, group caseGroup, row govform.ClaimRow, driverID *uuid.UUID) (ExportLine, error) {
	payload := ClaimLinePayload{
		Cells:     row.Cells,
		DriverID:  driverID,
		Direction: row.Direction,
		LegSeq:    row.LegSeq,
	}
	payload.Cells[0] = ""
	payload.Cells[6] = ""

	serviceDateROC, err := rocdate.ToROC(row.ServiceDate)
	if err != nil {
		return ExportLine{}, fmt.Errorf("convert service date to ROC: %w", err)
	}
	return ExportLine{
		LineNo:           lineNo,
		CaseID:           group.caseID,
		NationalIDMasked: group.nationalIDMasked,
		ServiceDateROC:   serviceDateROC,
		Payload:          payload,
	}, nil
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
	return "產生申報檔案失敗"
}

func (s *GovClaimService) loadImmutableFile(ctx context.Context, jobID, caseID uuid.UUID) ([]byte, error) {
	if store, ok := s.store.(ImmutableExportFileStore); ok {
		content, err := store.LoadExportFile(ctx, jobID, caseID)
		if err != nil {
			return nil, fmt.Errorf("load immutable export file: %w", err)
		}
		if len(content) > 0 {
			return content, nil
		}
	}
	// 相容尚未具備檔案內容欄位的舊匯出資料；新匯出一律由 immutable store 提供。
	return s.renderSnapshot(ctx, jobID, caseID)
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

type dataGapTally struct {
	order []string
	byKey map[string]*ClaimDataGap
}

func newDataGapTally() *dataGapTally {
	return &dataGapTally{byKey: make(map[string]*ClaimDataGap)}
}

func (t *dataGapTally) add(group caseGroup, reason string, count int) {
	key := group.caseID.String() + "|" + reason
	if existing, ok := t.byKey[key]; ok {
		existing.Count += count
		return
	}
	t.byKey[key] = &ClaimDataGap{CaseID: group.caseID, CaseName: group.caseName, Reason: reason, Count: count}
	t.order = append(t.order, key)
}

func (t *dataGapTally) list() []ClaimDataGap {
	result := make([]ClaimDataGap, 0, len(t.order))
	for _, key := range t.order {
		result = append(result, *t.byKey[key])
	}
	return result
}
