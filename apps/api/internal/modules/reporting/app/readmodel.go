package app

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/govform"
)

// 本檔是 reporting 的查詢投影（read model）。報表與檢核只讀不寫，這些型別沒有
// 獨立於查詢結果之外的持久化形狀，因此由 app 擁有、infra 直接掃描進來，不再多
// 一層等價的 row struct。它們一律不帶 struct tag，對外形狀由 transport 決定。

// VehicleTripTrend 代表車輛趟數趨勢資料。
type VehicleTripTrend struct {
	VehicleName string
	PlateNo     string
	TripCount   int
}

// IncompleteCase 代表資料不完整之個案。
type IncompleteCase struct {
	ID   uuid.UUID
	Name string
}

// UnresolvedConflict 代表未裁決衝突之搭乘紀錄。
type UnresolvedConflict struct {
	RideID      uuid.UUID
	CaseName    string
	ServiceDate time.Time
}

// ReportVehicleItem 代表報表車輛基礎資料。
type ReportVehicleItem struct {
	ID          uuid.UUID
	PlateNo     string
	DisplayName string
	Region      string
}

// ReportTripSummaryCaseRow 代表個案趟數統計資料。
type ReportTripSummaryCaseRow struct {
	CaseID        uuid.UUID
	CaseCode      string
	CaseName      string
	OutboundCount int
	InboundCount  int
	TotalCount    int
}

// ReportVehicleTripSummary 代表單一車輛趟數彙總資料。
type ReportVehicleTripSummary struct {
	Vehicle ReportVehicleItem
	Rows    []ReportTripSummaryCaseRow
}

// ReportHsinchuScheduleRow 代表新竹接送時刻表查詢列。
type ReportHsinchuScheduleRow struct {
	Direction   string
	RunNo       int16
	CaseCode    string
	CaseName    string
	Note        *string
	DepartTime  string
	ArriveTime  *string
	HomeAddress string
	SiteAddress string
	VehicleName string
	SiteName    string
}

// Renderer 產生報表檔案位元組，讓 app 不需認識任何試算表 SDK。
type Renderer interface {
	RenderTripSummary(periodYM string, vehicles []TripSummaryVehicle) ([]byte, error)
	RenderHsinchuSchedule(outbound, inbound []HsinchuScheduleItem) ([]byte, error)
	RenderGovClaim(rows []govform.ClaimRow) ([]byte, error)
}
