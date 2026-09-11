package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// RegionClaimInput 代表依區域批次建立政府申報匯出的輸入條件。
type RegionClaimInput struct {
	Regions       []string
	PeriodYMs     []string
	CreatedBy     uuid.UUID
	CreatedByName string
	ActorRole     string
}

// RegionClaimMonthResult 是單一月份的批次匯出結果。
// Job 為零值代表該月份沒有產出（原因記在 ErrorMessage），其餘月份仍照常回傳。
type RegionClaimMonthResult struct {
	PeriodYM     string
	Job          GovClaimJob
	Succeeded    bool
	ErrorMessage string
}

// RegionClaimResult 是整批依區域匯出的結果。
type RegionClaimResult struct {
	Regions   []string
	PeriodYMs []string
	CaseCount int
	Months    []RegionClaimMonthResult
	// SucceededJobIDs 供前端組出批次下載連結。
	SucceededJobIDs []uuid.UUID
	TotalFiles      int
}

// RegionClaimService 依區域展開個案後，逐月建立政府申報匯出工作。
//
// 逐月各建立一個 export_job 而不是把跨月檔案塞進同一個 job：export_job_files 有
// UNIQUE(job_id, case_id)，同一位個案的 5 月檔與 7 月檔在同一個 job 底下會直接撞唯一鍵；
// 而且 export_jobs.period_ym 是單月欄位、逐案下載路徑也以「一 job 一月」為前提。
// 逐月獨立另有一個好處：某個月查無資料或產檔失敗時，其餘月份仍拿得到檔案。
type RegionClaimService struct {
	scope  ClaimCaseResolver
	claims *GovClaimService
}

// NewRegionClaimService 建立 RegionClaimService 實例。
func NewRegionClaimService(scope ClaimCaseResolver, claims *GovClaimService) *RegionClaimService {
	return &RegionClaimService{scope: scope, claims: claims}
}

// CreateRegionClaimJobs 依區域解析個案後逐月產生申報檔。
func (s *RegionClaimService) CreateRegionClaimJobs(ctx context.Context, input RegionClaimInput) (RegionClaimResult, error) {
	if len(input.Regions) == 0 {
		return RegionClaimResult{}, ErrRegionsRequired
	}
	months, err := ParseClaimMonths(input.PeriodYMs)
	if err != nil {
		return RegionClaimResult{}, err
	}

	cases, err := s.scope.ListCasesByRegions(ctx, input.Regions)
	if err != nil {
		return RegionClaimResult{}, fmt.Errorf("list cases by regions: %w", err)
	}
	// 這幾個區域底下一位可申報的個案都沒有，是條件下錯而不是「這個月沒資料」，
	// 直接擋在建立任何工作之前，不留一串必然失敗的歷史紀錄。
	if len(cases) == 0 {
		return RegionClaimResult{}, ErrNoExportData
	}

	caseIDs := make([]uuid.UUID, 0, len(cases))
	for _, c := range cases {
		caseIDs = append(caseIDs, c.ID)
	}

	result := RegionClaimResult{
		Regions:   append([]string(nil), input.Regions...),
		CaseCount: len(caseIDs),
	}

	// 分別追蹤失敗原因的分類，讓「整批都失敗」的情況能依實際原因回應，
	// 而不是一律說成「沒有可申報的資料」：混車衝突未裁決該說清楚要先去裁決，
	// 內部錯誤（DB 故障等）更不該偽裝成「查無資料」讓使用者誤以為條件選錯。
	var (
		hasPrecheckBlocked bool
		firstInternalErr   error
	)

	for _, month := range months {
		result.PeriodYMs = append(result.PeriodYMs, month.PeriodYM)

		job, err := s.claims.CreateGovClaimJob(ctx, CreateGovClaimInput{
			PeriodYM: month.PeriodYM,
			CaseIDs:  caseIDs,
			// 一次數十份檔案沒有逐案點下載的道理，批次模式固定產壓縮檔。
			Mode:          GovClaimModeZip,
			CreatedBy:     input.CreatedBy,
			CreatedByName: input.CreatedByName,
			ActorRole:     input.ActorRole,
			Scope: ExportScopeSnapshot{
				Type:    ExportScopeTypeRegion,
				Regions: append([]string(nil), input.Regions...),
			},
		})
		if err != nil {
			// 單月失敗不中斷其餘月份：使用者要的是先拿到報得出來的月份。
			// 失敗原因已由 CreateGovClaimJob 寫進 export_jobs.error_message，
			// 但此處先前完全沒有伺服器端 log，出問題時只能從資料庫欄位回推，
			// 補上 slog 讓維運能直接從 log 找到失敗月份與原因。
			slog.Error("region_claim_month_failed",
				slog.String("period_ym", month.PeriodYM),
				slog.Any("regions", input.Regions),
				slog.String("error", err.Error()),
			)
			switch {
			case errors.Is(err, ErrPrecheckBlocked):
				hasPrecheckBlocked = true
			case errors.Is(err, ErrNoExportData):
				// 真的沒有資料，不影響整批分類判斷。
			default:
				if firstInternalErr == nil {
					firstInternalErr = err
				}
			}
			result.Months = append(result.Months, RegionClaimMonthResult{
				PeriodYM:     month.PeriodYM,
				ErrorMessage: monthFailureMessage(err),
			})
			continue
		}

		result.Months = append(result.Months, RegionClaimMonthResult{
			PeriodYM:  month.PeriodYM,
			Job:       job,
			Succeeded: true,
		})
		result.SucceededJobIDs = append(result.SucceededJobIDs, job.ID)
		result.TotalFiles += len(job.Files)
	}

	if len(result.SucceededJobIDs) == 0 {
		switch {
		case firstInternalErr != nil:
			// 含有非「沒資料／混車衝突」的內部錯誤時，不能說成「沒有資料」，
			// 讓呼叫端依一般錯誤處理（500），並保留其中一個具體原因供 log 對照。
			return RegionClaimResult{}, fmt.Errorf("create region claim jobs: %w", firstInternalErr)
		case hasPrecheckBlocked:
			// 全部月份都是因為未裁決的混車衝突被擋下，屬於「資料檢核未通過」而非「沒有資料」。
			return RegionClaimResult{}, ErrPrecheckBlocked
		default:
			// 每一個月份都是真的沒有可申報資料，整批視為查無資料。
			return RegionClaimResult{}, ErrNoExportData
		}
	}
	return result, nil
}

// monthFailureMessage 把單月失敗轉成使用者看得懂的原因；內部錯誤細節不外洩。
func monthFailureMessage(err error) string {
	switch {
	case errors.Is(err, ErrNoExportData):
		return "該月份沒有可申報的資料"
	case errors.Is(err, ErrPrecheckBlocked):
		return "該月份存在未裁決的混車衝突，需先完成裁決"
	default:
		return "產生申報檔案失敗"
	}
}
