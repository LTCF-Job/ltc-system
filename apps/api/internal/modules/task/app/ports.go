package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ScheduleLeg 是排班中的單一趟次。
type ScheduleLeg struct {
	LegSeq     int16
	Direction  string
	DepartTime string
	VehicleID  *uuid.UUID
}

// ActiveSchedule 是月結作業展開搭乘日曆所需的個案排班資訊。
type ActiveSchedule struct {
	CaseID        uuid.UUID
	CaseName      string
	Region        string
	ClaimEndDate  *time.Time
	SiteID        uuid.UUID
	SiteOpenDays  []int16
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Weekdays      []int16
	TripPattern   int16
	Legs          []ScheduleLeg
}

// TaskStore 定義月結與缺報檢核所需的搭乘紀錄查詢。
type TaskStore interface {
	GetReportedRideSlotsInRange(ctx context.Context, start, end time.Time) ([]ReportedRideSlotOnDate, error)
	GetMonthEndRideStats(ctx context.Context, start, end time.Time) (MonthEndRideStats, error)
}

// MonthScheduleReader 提供整月有效排班，由擁有個案能力的模組實作。
type MonthScheduleReader interface {
	GetActiveSchedulesForMonth(ctx context.Context, year, month int, region string) ([]ActiveSchedule, error)
}

// HolidayMapReader 提供排除假日所需的日期對照表。
type HolidayMapReader interface {
	GetHolidayMap(ctx context.Context, year, month int, region string) (map[string]bool, error)
}

// Notifier 定義月結與缺報告警的通知發送邊界。
type Notifier interface {
	SendNotification(ctx context.Context, topic, subject, body string) error
}

// ReportedRideSlotOnDate 代表已回報趟次之辨識鍵，含服務日期以避免跨日誤判。
type ReportedRideSlotOnDate struct {
	CaseID      uuid.UUID
	ServiceDate time.Time
	LegSeq      int16
}

// MonthEndRideStats 代表月統計搭乘資料。
type MonthEndRideStats struct {
	TotalRides      int
	BoardedRides    int
	UnreportedRides int
	ConflictCount   int
}
