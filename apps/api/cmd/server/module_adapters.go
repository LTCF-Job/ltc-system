package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/rocdate"
	caregiverapp "ltc-system/apps/api/internal/modules/caregiver/app"
	importapp "ltc-system/apps/api/internal/modules/caseimport/app"
	caseapp "ltc-system/apps/api/internal/modules/casemgmt/app"
	caseinfra "ltc-system/apps/api/internal/modules/casemgmt/infra"
	drapp "ltc-system/apps/api/internal/modules/driverreport/app"
	masterapp "ltc-system/apps/api/internal/modules/masterdata/app"
	masterinfra "ltc-system/apps/api/internal/modules/masterdata/infra"
	opsapp "ltc-system/apps/api/internal/modules/ops/app"
	rideapp "ltc-system/apps/api/internal/modules/ride/app"
	taskapp "ltc-system/apps/api/internal/modules/task/app"
)

// 跨模組協作一律走「消費者宣告 port、composition root 注入」：以下 adapter 把某個
// 模組的查詢結果轉成消費模組自己的型別，任一模組都不直接 import 另一個模組。

// caseSiteFinder 讓 casemgmt 驗證個案交通偏好的單位是否同區。
type caseSiteFinder struct{ repo *masterinfra.SiteRepository }

func (a caseSiteFinder) GetByID(ctx context.Context, id uuid.UUID) (*caseapp.SiteRef, error) {
	s, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &caseapp.SiteRef{ID: s.ID, Region: s.Region}, nil
}

// rideDriverResolver 讓 ride 由姓名或當日車輛推導司機。
type rideDriverResolver struct{ repo *masterinfra.DriverRepository }

func (a rideDriverResolver) GetByNameNormalized(ctx context.Context, nameNorm string) (*rideapp.DriverRef, error) {
	d, err := a.repo.GetByNameNormalized(ctx, nameNorm)
	if err != nil {
		if errors.Is(err, masterapp.ErrDriverNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if d == nil {
		return nil, nil
	}
	return &rideapp.DriverRef{ID: d.ID, Name: d.Name}, nil
}

func (a rideDriverResolver) ListDriversForVehicleOnDate(ctx context.Context, vehicleID uuid.UUID, serviceDate time.Time) ([]rideapp.DriverRef, error) {
	list, err := a.repo.ListDriversForVehicleOnDate(ctx, vehicleID, serviceDate)
	if err != nil {
		return nil, err
	}
	refs := make([]rideapp.DriverRef, 0, len(list))
	for _, d := range list {
		refs = append(refs, rideapp.DriverRef{ID: d.ID, Name: d.Name})
	}
	return refs, nil
}

// rideScheduleReader 讓 ride 取得比對回報所需的當日排班。
type rideScheduleReader struct{ repo *caseinfra.CaseRepository }

func (a rideScheduleReader) GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*rideapp.CaseSchedule, error) {
	s, err := a.repo.GetActiveScheduleForCaseOnDate(ctx, caseID, serviceDate)
	if err != nil || s == nil {
		return nil, err
	}
	legs := make([]rideapp.ScheduleLeg, 0, len(s.Legs))
	for _, l := range s.Legs {
		legs = append(legs, rideapp.ScheduleLeg{LegSeq: l.LegSeq, Direction: l.Direction, DepartTime: l.DepartTime, VehicleID: l.VehicleID})
	}
	return &rideapp.CaseSchedule{ID: s.ID, CaseID: s.CaseID, SiteID: s.SiteID, TripPattern: s.TripPattern, Legs: legs}, nil
}

// rideMissingReportProvider 讓 ride 的異常集中清單取得整月未回報趟次，不觸發告警通知。
type rideMissingReportProvider struct{ svc *taskapp.TaskService }

func (a rideMissingReportProvider) ListMissingForMonth(ctx context.Context, year, month int, region string) ([]rideapp.MissingRide, error) {
	items, err := a.svc.ListMissingReportsForMonth(ctx, year, month, region)
	if err != nil {
		return nil, err
	}
	out := make([]rideapp.MissingRide, 0, len(items))
	for _, item := range items {
		serviceDate, err := rocdate.ParseDate(item.ServiceDate)
		if err != nil {
			return nil, fmt.Errorf("parse missing ride service date: %w", err)
		}
		out = append(out, rideapp.MissingRide{
			CaseID:      item.CaseID,
			CaseName:    item.CaseName,
			ServiceDate: serviceDate,
			LegSeq:      item.LegSeq,
		})
	}
	return out, nil
}

// taskScheduleReader 讓 task 的月結作業取得整月有效排班。
type taskScheduleReader struct{ repo *caseinfra.CaseRepository }

func (a taskScheduleReader) GetActiveSchedulesForMonth(ctx context.Context, year, month int, region string) ([]taskapp.ActiveSchedule, error) {
	list, err := a.repo.GetActiveSchedulesForMonth(ctx, year, month, region)
	if err != nil {
		return nil, err
	}
	out := make([]taskapp.ActiveSchedule, 0, len(list))
	for _, s := range list {
		legs := make([]taskapp.ScheduleLeg, 0, len(s.Legs))
		for _, l := range s.Legs {
			legs = append(legs, taskapp.ScheduleLeg{LegSeq: l.LegSeq, Direction: l.Direction, DepartTime: l.DepartTime, VehicleID: l.VehicleID})
		}
		out = append(out, taskapp.ActiveSchedule{
			CaseID: s.CaseID, CaseName: s.CaseName, Region: s.Region,
			ClaimEndDate: s.ClaimEndDate, SiteID: s.SiteID,
			SiteOpenDays: s.SiteOpenDays, EffectiveFrom: s.EffectiveFrom, EffectiveTo: s.EffectiveTo,
			Weekdays: s.Weekdays, TripPattern: s.TripPattern, Legs: legs,
		})
	}
	return out, nil
}

// opsDriverLister 讓 ops 的出勤月報取得司機清單。
type opsDriverLister struct{ repo *masterinfra.DriverRepository }

func (a opsDriverLister) List(ctx context.Context, region, q string, page, pageSize int) ([]opsapp.DriverRef, int64, error) {
	list, total, err := a.repo.List(ctx, region, q, "", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]opsapp.DriverRef, 0, len(list))
	for _, d := range list {
		out = append(out, opsapp.DriverRef{ID: d.ID, Name: d.Name, Region: d.Region})
	}
	return out, total, nil
}

func (a opsDriverLister) ListAllActive(ctx context.Context) ([]opsapp.DriverRef, error) {
	list, err := a.repo.ListAllActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]opsapp.DriverRef, 0, len(list))
	for _, d := range list {
		out = append(out, opsapp.DriverRef{ID: d.ID, Name: d.Name, Region: d.Region})
	}
	return out, nil
}

// opsVehicleLister 讓 ops 的維修紀錄取得車輛清單。
type opsVehicleLister struct {
	repo *masterinfra.VehicleRepository
}

func (a opsVehicleLister) List(ctx context.Context, region, q string, page, pageSize int) ([]opsapp.VehicleRef, int64, error) {
	list, total, err := a.repo.List(ctx, masterapp.VehicleFilter{Region: region, Q: q}, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]opsapp.VehicleRef, 0, len(list))
	for _, v := range list {
		out = append(out, opsapp.VehicleRef{ID: v.ID, DisplayName: v.DisplayName, PlateNo: v.PlateNo})
	}
	return out, total, nil
}

func (a opsVehicleLister) ListAll(ctx context.Context) ([]opsapp.VehicleRef, error) {
	list, err := a.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]opsapp.VehicleRef, 0, len(list))
	for _, v := range list {
		out = append(out, opsapp.VehicleRef{ID: v.ID, DisplayName: v.DisplayName, PlateNo: v.PlateNo})
	}
	return out, nil
}

// importSiteLookup 讓 caseimport 以名稱或區域比對單位。
type importSiteLookup struct{ repo *masterinfra.SiteRepository }

func (a importSiteLookup) GetByName(ctx context.Context, name string) (*importapp.SiteRef, error) {
	s, err := a.repo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, masterapp.ErrSiteNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return &importapp.SiteRef{ID: s.ID, Name: s.Name}, nil
}

func (a importSiteLookup) List(ctx context.Context, region string, page, pageSize int) ([]importapp.SiteRef, error) {
	list, _, err := a.repo.List(ctx, region, "", "", page, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]importapp.SiteRef, 0, len(list))
	for _, s := range list {
		out = append(out, importapp.SiteRef{ID: s.ID, Name: s.Name})
	}
	return out, nil
}

// importVehicleLookup 讓 caseimport 以顯示名稱比對車輛。
type importVehicleLookup struct {
	repo *masterinfra.VehicleRepository
}

func (a importVehicleLookup) GetByDisplayName(ctx context.Context, displayName string) (*importapp.VehicleRef, error) {
	v, err := a.repo.GetByDisplayName(ctx, displayName)
	if err != nil {
		if errors.Is(err, masterapp.ErrVehicleNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return &importapp.VehicleRef{ID: v.ID}, nil
}

// caseRegistrar 讓 caseimport 透過 casemgmt 寫入個案主檔。
type caseRegistrar struct{ svc *caseapp.CaseService }

func (a caseRegistrar) CreateCase(ctx context.Context, in importapp.NewCase, actor importapp.Actor) (uuid.UUID, error) {
	entity, err := a.svc.CreateCase(ctx, caseapp.CreateCaseRequest{
		Name: in.Name, NationalID: in.NationalID,
		HouseholdType: in.HouseholdType, Gender: in.Gender, BirthDate: in.BirthDate,
		CareContactRole: in.CareContactRole, CareContactName: in.CareContactName,
		RegisteredAddress: in.RegisteredAddress, HomeAddress: in.HomeAddress, Region: in.Region,
		ServiceCategory:  intPointerOrNil(in.ServiceCategory),
		ServiceUsageType: intPointerOrNil(in.ServiceUsageType), Status: in.Status, Remarks: in.Remarks,
	}, actor.ActorID, actor.ActorRole, actor.IPAddress, actor.UserAgent)
	if err != nil {
		return uuid.Nil, err
	}
	return entity.ID, nil
}

func (a caseRegistrar) RecordSkipped(ctx context.Context, row importapp.CaseImportSkippedRow, actor importapp.Actor) {
	a.svc.RecordSkippedCaseImport(ctx, caseapp.CaseImportSkippedRow{
		RowID: row.RowID, RowIndex: row.RowIndex, CaseName: row.CaseName, Reasons: row.Reasons, RawValues: row.RawValues,
	}, actor.ActorID, actor.ActorRole, actor.IPAddress, actor.UserAgent)
}

// intPointerOrNil 匯入範本目前沒有服務類別／服務使用類型欄位，一律視為未提供；
// 0 是 Go 的零值而非使用者填的真實資料，不得當成合法值寫入。
func intPointerOrNil(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

// caregiverSiteLookup 讓 caregiver 匯入時以名稱比對單位。
type caregiverSiteLookup struct{ repo *masterinfra.SiteRepository }

func (a caregiverSiteLookup) GetByName(ctx context.Context, name string) (*caregiverapp.SiteRef, error) {
	s, err := a.repo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, masterapp.ErrSiteNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return &caregiverapp.SiteRef{ID: s.ID, Name: s.Name}, nil
}

// driverReportCaseLookup 讓 driverreport 以姓名相似度推薦欄位要對應的個案。
type driverReportCaseLookup struct{ repo *caseinfra.CaseRepository }

func (a driverReportCaseLookup) ListActiveCases(ctx context.Context) ([]drapp.CaseRef, error) {
	list, err := a.repo.ListNameIndex(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]drapp.CaseRef, 0, len(list))
	for _, c := range list {
		out = append(out, drapp.CaseRef{ID: c.ID.String(), Name: c.Name, NameNormalized: c.NameNormalized})
	}
	return out, nil
}

// driverReportDriverResolver 讓 driverreport 由匯報表上的駕駛人姓名比對司機主檔。
type driverReportDriverResolver struct{ repo *masterinfra.DriverRepository }

func (a driverReportDriverResolver) GetByNameNormalized(ctx context.Context, nameNorm string) (*drapp.DriverRef, error) {
	d, err := a.repo.GetByNameNormalized(ctx, nameNorm)
	if err != nil {
		if errors.Is(err, masterapp.ErrDriverNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if d == nil {
		return nil, nil
	}
	return &drapp.DriverRef{ID: d.ID, Name: d.Name}, nil
}

// driverReportAttendanceRegistrar 讓 driverreport 比對到司機時，同步該司機當天的出勤登記。
type driverReportAttendanceRegistrar struct{ svc *opsapp.AttendanceService }

func (a driverReportAttendanceRegistrar) SyncFromImport(ctx context.Context, driverID uuid.UUID, serviceDate time.Time) error {
	return a.svc.SyncFromImport(ctx, driverID, serviceDate)
}

// driverReportRideIngestor 讓 driverreport 把每日匯報交給 ride 展開為搭乘紀錄。
type driverReportRideIngestor struct{ svc *rideapp.RideService }

func (a driverReportRideIngestor) IngestSubmission(ctx context.Context, formID, vehicleID uuid.UUID, s drapp.Submission) (int, error) {
	return a.svc.IngestSubmission(ctx, formID, vehicleID, rideapp.ProcessSubmissionRequest{
		ServiceDate: s.ServiceDate,
		SubmittedAt: s.SubmittedAt,
		DriverRaw:   s.DriverRaw,
		DriverID:    s.DriverID,
		Remark:      s.Remark,
		Answers:     s.Answers,
	})
}

func (a driverReportRideIngestor) ClearImportedDates(ctx context.Context, formID uuid.UUID, dates []time.Time) (int, error) {
	return a.svc.ClearImportedDates(ctx, formID, dates)
}

func (a driverReportRideIngestor) BackfillColumn(ctx context.Context, formID, vehicleID uuid.UUID, columnHeader string, columnIndex int, caseID uuid.UUID, legSeq int16) (int, error) {
	return a.svc.BackfillColumn(ctx, formID, vehicleID, columnHeader, columnIndex, caseID, legSeq)
}

func (a driverReportRideIngestor) ListSubmissionsForForms(ctx context.Context, formIDs []uuid.UUID) ([]drapp.SubmissionAnswerRow, error) {
	rows, err := a.svc.ListSubmissionsForForms(ctx, formIDs)
	if err != nil {
		return nil, err
	}
	out := make([]drapp.SubmissionAnswerRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, drapp.SubmissionAnswerRow{
			SubmissionID: r.SubmissionID,
			FormID:       r.FormID,
			FormTitle:    r.FormTitle,
			VehicleName:  r.VehicleName,
			ServiceDate:  r.ServiceDate,
			Answers:      r.Answers,
		})
	}
	return out, nil
}

func (a driverReportRideIngestor) ListUnmatchedDriverSubmissions(ctx context.Context) ([]drapp.UnmatchedDriverSubmission, error) {
	rows, err := a.svc.ListUnmatchedDriverSubmissions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]drapp.UnmatchedDriverSubmission, 0, len(rows))
	for _, r := range rows {
		out = append(out, drapp.UnmatchedDriverSubmission{
			SubmissionID:  r.SubmissionID,
			FormID:        r.FormID,
			FormTitle:     r.FormTitle,
			VehicleName:   r.VehicleName,
			ServiceDate:   r.ServiceDate,
			DriverNameRaw: r.DriverNameRaw,
		})
	}
	return out, nil
}

func (a driverReportRideIngestor) BackfillDriver(ctx context.Context, driverNameRaw string, driverID uuid.UUID) (int, []time.Time, error) {
	return a.svc.BackfillDriver(ctx, driverNameRaw, driverID)
}

func (a driverReportRideIngestor) ListImportedMonths(ctx context.Context) ([]drapp.ImportedMonth, error) {
	months, err := a.svc.ListImportedMonths(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]drapp.ImportedMonth, 0, len(months))
	for _, m := range months {
		out = append(out, drapp.ImportedMonth{
			FormID:          m.FormID,
			YearMonth:       m.YearMonth,
			SubmissionCount: m.SubmissionCount,
			LastImportedAt:  m.LastImportedAt,
		})
	}
	return out, nil
}

func (a driverReportRideIngestor) ListSubmissionsForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]drapp.MonthSubmissionDetail, error) {
	rows, err := a.svc.ListSubmissionsForFormMonth(ctx, formID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	out := make([]drapp.MonthSubmissionDetail, 0, len(rows))
	for _, r := range rows {
		out = append(out, drapp.MonthSubmissionDetail{
			ServiceDate:   r.ServiceDate,
			DriverNameRaw: r.DriverNameRaw,
			Remark:        r.Remark,
			Answers:       r.Answers,
		})
	}
	return out, nil
}

func (a driverReportRideIngestor) ListRideEntriesForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]drapp.MonthRideEntry, error) {
	rows, err := a.svc.ListRideEntriesForFormMonth(ctx, formID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	out := make([]drapp.MonthRideEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, drapp.MonthRideEntry{
			CaseID:      r.CaseID,
			CaseName:    r.CaseName,
			ServiceDate: r.ServiceDate,
			LegSeq:      r.LegSeq,
			Reported:    r.Reported,
			DriverID:    r.DriverID,
			DriverName:  r.DriverName,
			VehicleID:   r.VehicleID,
		})
	}
	return out, nil
}

// caseDuplicateFinder 讓 caseimport 於 dry-run 階段透過 casemgmt 查重。
type caseDuplicateFinder struct{ svc *caseapp.CaseService }

func (a caseDuplicateFinder) FindDuplicate(ctx context.Context, nationalID, name string) (*importapp.DuplicateRef, error) {
	found, err := a.svc.FindPossibleDuplicate(ctx, nationalID, name)
	if err != nil || found == nil {
		return nil, err
	}
	return &importapp.DuplicateRef{CaseID: found.ID, CaseName: found.Name}, nil
}
