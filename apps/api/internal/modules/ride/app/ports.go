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
	GetPrimaryDriverForVehicleOnDate(ctx context.Context, vehicleID uuid.UUID, serviceDate time.Time) (*DriverRef, error)
}

// RideRecordStore 定義表單提交、來源列與搭乘紀錄的讀寫邊界。
type RideRecordStore interface {
	GetFormBySecret(ctx context.Context, secret string) (uuid.UUID, uuid.UUID, error)
	GetFormColumns(ctx context.Context, formID uuid.UUID) ([]FormColumn, error)
	SaveFormSubmission(ctx context.Context, formID uuid.UUID, serviceDate, submittedAt time.Time, driverNameRaw string, driverID *uuid.UUID, source string, payload map[string]interface{}, issueText string) (uuid.UUID, error)
	InsertRideSource(ctx context.Context, submissionID, caseID uuid.UUID, serviceDate time.Time, legSeq int16, vehicleID uuid.UUID, driverID *uuid.UUID, reported string, colIdx int) error
	GetRideRecordForSlot(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) (*RideRecord, error)
	UpsertRideRecord(ctx context.Context, rec *RideRecord) error
	CorrectRideRecord(ctx context.Context, rideID uuid.UUID, effectiveStatus *string, vehicleID, driverID *uuid.UUID, departTimeOverride *string, durationMinOverride *int16, notClaimedAA09 *bool, reason *string, operatorID uuid.UUID) error
}
