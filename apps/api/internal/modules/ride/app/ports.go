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

// RideSlot 是一筆搭乘紀錄的唯一座標。
type RideSlot struct {
	CaseID      uuid.UUID
	ServiceDate time.Time
	LegSeq      int16
}

// ImportedMonth 是某份匯報表在某個月已匯入的提交統計。
type ImportedMonth struct {
	FormID          uuid.UUID
	YearMonth       string
	SubmissionCount int
	LastImportedAt  time.Time
}

// RideSourceRow 是單一 slot 已寫入的一筆回報來源，混車合併以此為輸入。
type RideSourceRow struct {
	VehicleID   uuid.UUID
	DriverID    *uuid.UUID
	Reported    string
	SubmittedAt time.Time
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
	Code          string
	Name          string
	Region        string
	TripPattern   int16
	Weekdays      []int16
	SiteOpenDays  []int16
	ClaimEndDate  *time.Time
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Legs          []CalendarLeg
}

// RideRecordStore 定義表單提交、來源列與搭乘紀錄的讀寫邊界。
type RideRecordStore interface {
	GetFormColumns(ctx context.Context, formID uuid.UUID) ([]FormColumn, error)
	ListRideSourcesForSlot(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) ([]RideSourceRow, error)
	ListCalendarCases(ctx context.Context, start, end time.Time, region, keyword string) ([]CalendarCase, error)
	ListRideRecordsInRange(ctx context.Context, start, end time.Time, region, keyword string) ([]RideRecord, error)
	SaveFormSubmission(ctx context.Context, formID uuid.UUID, serviceDate, submittedAt time.Time, driverNameRaw string, driverID *uuid.UUID, source string, payload map[string]interface{}, issueText string, anomalyFlags []string) (uuid.UUID, error)
	InsertRideSource(ctx context.Context, submissionID, caseID uuid.UUID, serviceDate time.Time, legSeq int16, vehicleID uuid.UUID, driverID *uuid.UUID, reported string, colIdx int) error
	ListRideSourceSlotsForForm(ctx context.Context, formID uuid.UUID, dates []time.Time) ([]RideSlot, error)
	DeleteFormSubmissions(ctx context.Context, formID uuid.UUID, dates []time.Time) (int, error)
	ListImportedMonths(ctx context.Context) ([]ImportedMonth, error)
	DeleteDerivedRideRecord(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) error
	GetRideRecordForSlot(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) (*RideRecord, error)
	UpsertRideRecord(ctx context.Context, rec *RideRecord) error
	CorrectRideRecord(ctx context.Context, rideID uuid.UUID, effectiveStatus *string, vehicleID, driverID *uuid.UUID, departTimeOverride *string, durationMinOverride *int16, notClaimedAA09 *bool, reason *string, operatorID uuid.UUID) error
	GetRideRecordByID(ctx context.Context, id uuid.UUID) (*RideRecord, error)
	ResolveConflict(ctx context.Context, rideID, vehicleID uuid.UUID, driverID *uuid.UUID, note *string, operatorID uuid.UUID) (bool, error)
	ListPendingConflicts(ctx context.Context, start, end time.Time, keyword string, page, pageSize int) ([]ConflictRide, int64, error)
	ListImportErrorSubmissions(ctx context.Context, start, end time.Time, keyword string, page, pageSize int) ([]ImportErrorSubmission, int64, error)
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
	ListMissingForMonth(ctx context.Context, year, month int, region string) ([]MissingRide, error)
}
