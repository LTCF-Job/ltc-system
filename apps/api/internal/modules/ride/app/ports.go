package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CaseSchedule 是比對回報時需要的個案排班資訊。
type CaseSchedule struct {
	ID          uuid.UUID
	CaseID      uuid.UUID
	SiteID      uuid.UUID
	Weekdays    []int16
	TripPattern int16
	Legs        []ScheduleLeg
}

// ScheduleLeg 是排班中的單一趟次。
type ScheduleLeg struct {
	LegSeq     int16
	Direction  string
	DepartTime string
	VehicleID  *uuid.UUID
}

// DriverRef 是由姓名或當日車輛推導出的司機。
type DriverRef struct {
	ID   uuid.UUID
	Name string
}

// AuditEntry 是本模組寫入稽核日誌的內容。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData interface{}
	AfterData  interface{}
	IPAddress  *string
	UserAgent  *string
}

// AuditWriter 定義搭乘紀錄更正留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// ScheduleReader 提供比對回報所需的當日有效排班，由擁有個案能力的模組實作。
type ScheduleReader interface {
	GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*CaseSchedule, error)
}

// DriverResolver 由姓名或當日車輛推導司機，由擁有司機主檔的模組實作。
type DriverResolver interface {
	GetByNameNormalized(ctx context.Context, nameNorm string) (*DriverRef, error)
	ListDriversForVehicleOnDate(ctx context.Context, vehicleID uuid.UUID, serviceDate time.Time) ([]DriverRef, error)
}

// ImportedMonth 是某份匯報表在某個月已匯入的提交統計。
type ImportedMonth struct {
	FormID          uuid.UUID
	YearMonth       string
	SubmissionCount int
	LastImportedAt  time.Time
}

// MonthSubmissionDetail 是某份匯報表某個月一天的完整回報內容，供總覽頁鑽取查看
// 逐日原始資料，不需重新開啟原始檔案。
type MonthSubmissionDetail struct {
	ServiceDate   time.Time
	DriverNameRaw string
	Remark        string
	Answers       map[string]string
}

// MonthRideEntry 是某份匯報表某個月展開後的一筆個案搭乘紀錄，供總覽頁鑽取查看
// 這個月實際寫入了哪些個案、哪些趟次。
type MonthRideEntry struct {
	CaseID      uuid.UUID
	CaseName    string
	ServiceDate time.Time
	LegSeq      int16
	Reported    string
	DriverID    *uuid.UUID
	DriverName  string
	VehicleID   uuid.UUID
}

// RideSourceRow 是單一 slot 已寫入的一筆回報來源，混車合併以此為輸入。
type RideSourceRow struct {
	SourceID       uuid.UUID
	SubmissionID   uuid.UUID
	SourcePriority int
	VehicleID      uuid.UUID
	DriverID       *uuid.UUID
	Reported       string
	SubmittedAt    time.Time
}

// SubmissionAnswer 是某欄位在一筆既有回報中留下的原始儲存格文字，供欄位補綁定後
// 直接回填搭乘紀錄，不需重新上傳原始檔案。SubmittedAt 是這筆提交原本的上傳時間，
// 回填時仍要用它比對「同車同個案」是否衝突，不能當作是現在才發生的新回報。
type SubmissionAnswer struct {
	SubmissionID uuid.UUID
	ServiceDate  time.Time
	SubmittedAt  time.Time
	DriverID     *uuid.UUID
	Value        string
}

// SubmissionFull 是一筆既有回報的完整內容，供彙整待維護清單時比對哪些欄位這一列
// 「有回報」但仍待對應個案。
type SubmissionFull struct {
	SubmissionID uuid.UUID
	FormID       uuid.UUID
	FormTitle    string
	VehicleName  string
	ServiceDate  time.Time
	Answers      map[string]string
}

// UnmatchedDriverSubmission 是一筆駕駛人姓名比對不到司機主檔的既有回報。VehicleID 與
// Answers 供司機補綁定後，不需重新上傳原始檔案即可比照 IngestSubmission 逐欄重新比對寫入——
// 這一列在司機比對不到時完全沒有展開成 ride_sources（見「資料不完整」閘門），所以回填時
// 是從頭比對寫入，不是更新既有來源。
type UnmatchedDriverSubmission struct {
	SubmissionID  uuid.UUID
	FormID        uuid.UUID
	FormTitle     string
	VehicleID     uuid.UUID
	VehicleName   string
	ServiceDate   time.Time
	SubmittedAt   time.Time
	DriverNameRaw string
	Answers       map[string]string
}

// RowConflict 是一筆待處理的「同車同個案」衝突：這台車在同一 slot 已有生效資料，這次
// 上傳或回填的值與既有值不同（回報值或司機任一不同），需要使用者選擇要保留哪一筆，
// 在使用者選擇前不寫入 ride_sources。
type RowConflict struct {
	ID                   uuid.UUID
	FormID               uuid.UUID
	FormTitle            string
	VehicleID            uuid.UUID
	VehicleName          string
	CaseID               uuid.UUID
	CaseName             string
	ServiceDate          time.Time
	LegSeq               int16
	SourceColumnIndex    int
	PreviousSubmissionID uuid.UUID
	PreviousReported     string
	PreviousDriverID     *uuid.UUID
	PreviousDriverName   string
	PreviousSubmittedAt  time.Time
	NewSubmissionID      uuid.UUID
	NewReported          string
	NewDriverID          *uuid.UUID
	NewDriverName        string
	NewSubmittedAt       time.Time
	DetectedAt           time.Time
}

// RowConflictInput 是暫存一筆「同車同個案」衝突所需的欄位。
type RowConflictInput struct {
	FormID               uuid.UUID
	VehicleID            uuid.UUID
	CaseID               uuid.UUID
	ServiceDate          time.Time
	LegSeq               int16
	SourceColumnIndex    int
	PreviousSubmissionID uuid.UUID
	PreviousReported     string
	PreviousDriverID     *uuid.UUID
	PreviousSubmittedAt  time.Time
	NewSubmissionID      uuid.UUID
	NewReported          string
	NewDriverID          *uuid.UUID
	NewSubmittedAt       time.Time
}

// AppliedRowConflict 是裁決「採用新資料」時，重放寫入搭乘來源所需的欄位。
type AppliedRowConflict struct {
	CaseID            uuid.UUID
	ServiceDate       time.Time
	LegSeq            int16
	VehicleID         uuid.UUID
	SourceColumnIndex int
	NewSubmissionID   uuid.UUID
	NewReported       string
	NewDriverID       *uuid.UUID
	NewSubmittedAt    time.Time
}

// RideSourceSlot 是一筆搭乘來源所屬的「同車同個案同趟次」位置，供刪除來源後重算搭乘紀錄。
type RideSourceSlot struct {
	CaseID      uuid.UUID
	ServiceDate time.Time
	LegSeq      int16
	VehicleID   uuid.UUID
}

// CalendarLeg 是月曆表需要的排班趟次時段。
type CalendarLeg struct {
	LegSeq      int16
	Direction   string
	DepartTime  string
	VehicleID   *uuid.UUID
	VehicleName string
}

// CalendarCase 是月曆表一列所需的個案與其當期排班。
type CalendarCase struct {
	ID            uuid.UUID
	Name          string
	TripPattern   int16
	Weekdays      []int16
	ClaimEndDate  *time.Time
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Legs          []CalendarLeg
}

// PatchValue 表示 PATCH 欄位的三態：Present=false 代表保留、Present=true 且
// Value=nil 代表清除、Present=true 且 Value 非 nil 代表設定新值。
type PatchValue[T any] struct {
	Present bool
	Value   *T
}

// RideRecordStore 定義表單提交、來源列與搭乘紀錄的讀寫邊界。
type RideRecordStore interface {
	GetFormColumns(ctx context.Context, formID uuid.UUID) ([]FormColumn, error)
	ListRideSourcesForSlot(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) ([]RideSourceRow, error)
	ListCalendarCases(ctx context.Context, start, end time.Time, keyword string) ([]CalendarCase, error)
	ListRideRecordsInRange(ctx context.Context, start, end time.Time, keyword string) ([]RideRecord, error)
	SaveFormSubmission(ctx context.Context, formID uuid.UUID, serviceDate, submittedAt time.Time, driverNameRaw string, driverID *uuid.UUID, source string, payload map[string]interface{}, issueText string, anomalyFlags []string) (uuid.UUID, error)
	InsertRideSource(ctx context.Context, submissionID, caseID uuid.UUID, serviceDate time.Time, legSeq int16, vehicleID uuid.UUID, driverID *uuid.UUID, reported string, colIdx int, submittedAt time.Time) error
	ListSubmissionAnswersForColumn(ctx context.Context, formID uuid.UUID, columnHeader string) ([]SubmissionAnswer, error)
	ListImportedMonths(ctx context.Context) ([]ImportedMonth, error)
	// UpsertRideSourceRowConflict 暫存或更新一筆「同車同個案」衝突；同一 slot 只保留一筆
	// 未解決的衝突，第二次上傳到同一個未解決的衝突會直接更新其新值。
	UpsertRideSourceRowConflict(ctx context.Context, in RowConflictInput) (uuid.UUID, error)
	// ListPendingRowConflicts 取出目前所有尚未解決的「同車同個案」衝突。
	ListPendingRowConflicts(ctx context.Context) ([]RowConflict, error)
	// ResolveRowConflict 裁決一筆同車同個案衝突；resolved=false 代表已被他人裁決過。
	// useNew=true 時回傳需要重放寫入搭乘來源的欄位。
	ResolveRowConflict(ctx context.Context, conflictID uuid.UUID, useNew bool, operatorID uuid.UUID) (applied *AppliedRowConflict, resolved bool, err error)
	// ListRideSourceSlotsForSubmission 取出某筆提交紀錄展開出的搭乘來源 slot，供刪除前備妥重算範圍。
	ListRideSourceSlotsForSubmission(ctx context.Context, submissionID uuid.UUID) ([]RideSourceSlot, error)
	// DeleteSubmission 移除一筆提交紀錄；rowsAffected=0 代表該列不存在。
	DeleteSubmission(ctx context.Context, submissionID uuid.UUID) (rowsAffected int64, err error)
	// DeleteRowConflict 移除一筆尚未裁決的同車同個案衝突；rowsAffected=0 代表該列不存在或已被裁決。
	DeleteRowConflict(ctx context.Context, conflictID uuid.UUID) (rowsAffected int64, err error)
	// ListSubmissionsForFormMonth 取出某份匯報表在 [monthStart, monthEnd) 區間內的逐日原始回報，
	// 供總覽頁鑽取單一月份的完整內容。
	ListSubmissionsForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]MonthSubmissionDetail, error)
	// ListRideEntriesForFormMonth 取出某份匯報表在 [monthStart, monthEnd) 區間內展開後的個案搭乘紀錄，
	// 供總覽頁鑽取單一月份實際寫入了哪些個案與趟次。
	ListRideEntriesForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]MonthRideEntry, error)
	DeleteDerivedRideRecord(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) error
	GetRideRecordForSlot(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) (*RideRecord, error)
	UpsertRideRecord(ctx context.Context, rec *RideRecord) error
	CorrectRideRecord(ctx context.Context, rideID uuid.UUID, effectiveStatus PatchValue[string], vehicleID, driverID PatchValue[uuid.UUID], departTimeOverride PatchValue[string], durationMinOverride PatchValue[int16], notClaimedAA09 PatchValue[bool], reason PatchValue[string], operatorID uuid.UUID) error
	GetRideRecordByID(ctx context.Context, id uuid.UUID) (*RideRecord, error)
	ResolveConflict(ctx context.Context, rideID, vehicleID uuid.UUID, driverID *uuid.UUID, note *string, operatorID uuid.UUID) (bool, error)
	ListPendingConflicts(ctx context.Context, start, end time.Time, keyword string, page, pageSize int) ([]ConflictRide, int64, error)
	ListImportErrorSubmissions(ctx context.Context, start, end time.Time, keyword string, page, pageSize int) ([]ImportErrorSubmission, int64, error)
	// ListSubmissionsForForms 取出指定表單目前存在的所有回報列與其完整原始儲存格文字。
	ListSubmissionsForForms(ctx context.Context, formIDs []uuid.UUID) ([]SubmissionFull, error)
	// ListUnmatchedDriverSubmissions 取出目前駕駛人姓名比對不到司機主檔的既有回報。
	ListUnmatchedDriverSubmissions(ctx context.Context) ([]UnmatchedDriverSubmission, error)
	// UpdateSubmissionDriverID 回填某筆提交紀錄的司機。
	UpdateSubmissionDriverID(ctx context.Context, submissionID, driverID uuid.UUID) error
}

// CorrectionFingerprintStore 保存人工更正所依據的來源快照；由 composition root 以 optional port
// 擴充既有 store，避免離線 fake 必須同時實作新欄位。
type CorrectionFingerprintStore interface {
	SetCorrectionFingerprint(ctx context.Context, rideID uuid.UUID, fingerprint string) error
}

// CorrectionFingerprintingStore 將人工更正與來源快照以同一筆資料庫更新寫入，
// 避免更正成功後 fingerprint 寫入失敗，留下無法判斷來源版本的半完成狀態。
type CorrectionFingerprintingStore interface {
	CorrectRideRecordWithFingerprint(
		ctx context.Context,
		rideID uuid.UUID,
		effectiveStatus PatchValue[string],
		vehicleID, driverID PatchValue[uuid.UUID],
		departTimeOverride PatchValue[string],
		durationMinOverride PatchValue[int16],
		notClaimedAA09 PatchValue[bool],
		reason PatchValue[string],
		operatorID uuid.UUID,
		fingerprint string,
	) error
}

// ConflictRide 是一筆待裁決混車衝突。
type ConflictRide struct {
	ID          uuid.UUID
	CaseID      uuid.UUID
	CaseName    string
	ServiceDate time.Time
	LegSeq      int16
	Vehicles    []string
}

// ImportErrorSubmission 是一筆表單匯入異常紀錄。
type ImportErrorSubmission struct {
	ID            uuid.UUID
	ServiceDate   time.Time
	DriverNameRaw string
	AnomalyFlags  []string
	RawPayload    string
}

// MissingRide 是一筆應搭未回報之趟次，由 task 模組經 MissingReportProvider 提供。
type MissingRide struct {
	CaseID      uuid.UUID
	CaseName    string
	ServiceDate time.Time
	LegSeq      int16
	VehicleName string
}

// MissingReportProvider 提供整月未回報清單，由擁有排班/回報比對能力的模組實作。
type MissingReportProvider interface {
	ListMissingForMonth(ctx context.Context, year, month int) ([]MissingRide, error)
}
