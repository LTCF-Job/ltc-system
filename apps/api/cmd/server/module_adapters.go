package main

import (
	"context"
	"time"

	"github.com/google/uuid"
	caregiverapp "ltc-system/apps/api/internal/modules/caregiver/app"
	importapp "ltc-system/apps/api/internal/modules/caseimport/app"
	caseapp "ltc-system/apps/api/internal/modules/casemgmt/app"
	caseinfra "ltc-system/apps/api/internal/modules/casemgmt/infra"
	masterinfra "ltc-system/apps/api/internal/modules/masterdata/infra"
	opsapp "ltc-system/apps/api/internal/modules/ops/app"
	rideapp "ltc-system/apps/api/internal/modules/ride/app"
	taskapp "ltc-system/apps/api/internal/modules/task/app"
)

// 跨模組協作一律走「消費者宣告 port、composition root 注入」：以下 adapter 把某個
// 模組的查詢結果轉成消費模組自己的型別，任一模組都不直接 import 另一個模組。

// caseSiteFinder 讓 casemgmt 驗證個案交通偏好的據點是否同區。
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
		return nil, err
	}
	return &rideapp.DriverRef{ID: d.ID, Name: d.Name}, nil
}

func (a rideDriverResolver) GetPrimaryDriverForVehicleOnDate(ctx context.Context, vehicleID uuid.UUID, serviceDate time.Time) (*rideapp.DriverRef, error) {
	d, err := a.repo.GetPrimaryDriverForVehicleOnDate(ctx, vehicleID, serviceDate)
	if err != nil {
		return nil, err
	}
	return &rideapp.DriverRef{ID: d.ID, Name: d.Name}, nil
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
			CaseID: s.CaseID, CaseCode: s.CaseCode, CaseName: s.CaseName, Region: s.Region,
			ClaimStartDate: s.ClaimStartDate, ClaimEndDate: s.ClaimEndDate, SiteID: s.SiteID,
			SiteOpenDays: s.SiteOpenDays, EffectiveFrom: s.EffectiveFrom, EffectiveTo: s.EffectiveTo,
			Weekdays: s.Weekdays, TripPattern: s.TripPattern, Legs: legs,
		})
	}
	return out, nil
}

// opsDriverLister 讓 ops 的出勤月報取得司機清單。
type opsDriverLister struct{ repo *masterinfra.DriverRepository }

func (a opsDriverLister) List(ctx context.Context, region, q string, page, pageSize int) ([]opsapp.DriverRef, int64, error) {
	list, total, err := a.repo.List(ctx, region, q, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]opsapp.DriverRef, 0, len(list))
	for _, d := range list {
		out = append(out, opsapp.DriverRef{ID: d.ID, Name: d.Name, Code: d.Code, Region: d.Region})
	}
	return out, total, nil
}

// opsVehicleLister 讓 ops 的維修紀錄取得車輛清單。
type opsVehicleLister struct {
	repo *masterinfra.VehicleRepository
}

func (a opsVehicleLister) List(ctx context.Context, region, q string, page, pageSize int) ([]opsapp.VehicleRef, int64, error) {
	list, total, err := a.repo.List(ctx, region, q, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]opsapp.VehicleRef, 0, len(list))
	for _, v := range list {
		out = append(out, opsapp.VehicleRef{ID: v.ID, DisplayName: v.DisplayName, PlateNo: v.PlateNo})
	}
	return out, total, nil
}

// importSiteLookup 讓 caseimport 以名稱或區域比對據點。
type importSiteLookup struct{ repo *masterinfra.SiteRepository }

func (a importSiteLookup) GetByName(ctx context.Context, name string) (*importapp.SiteRef, error) {
	s, err := a.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &importapp.SiteRef{ID: s.ID, Name: s.Name}, nil
}

func (a importSiteLookup) List(ctx context.Context, region string, page, pageSize int) ([]importapp.SiteRef, error) {
	list, _, err := a.repo.List(ctx, region, "", page, pageSize)
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
		return nil, err
	}
	return &importapp.VehicleRef{ID: v.ID}, nil
}

// caseRegistrar 讓 caseimport 透過 casemgmt 寫入個案主檔。
type caseRegistrar struct{ svc *caseapp.CaseService }

func (a caseRegistrar) CreateCase(ctx context.Context, in importapp.NewCase, actor importapp.Actor) (uuid.UUID, error) {
	entity, err := a.svc.CreateCase(ctx, caseapp.CreateCaseRequest{
		Code: in.Code, Name: in.Name, NationalID: in.NationalID,
		HouseholdType: in.HouseholdType, Gender: in.Gender, BirthDate: in.BirthDate,
		CareContactRole: in.CareContactRole, CareContactName: in.CareContactName,
		RegisteredAddress: in.RegisteredAddress, HomeAddress: in.HomeAddress, Region: in.Region,
		ClaimStartDate: in.ClaimStartDate, ServiceCategory: in.ServiceCategory,
		ServiceUsageType: in.ServiceUsageType, Status: in.Status, Remarks: in.Remarks,
	}, actor.ActorID, actor.ActorRole, actor.IPAddress, actor.UserAgent)
	if err != nil {
		return uuid.Nil, err
	}
	return entity.ID, nil
}

func (a caseRegistrar) RecordSkipped(ctx context.Context, row importapp.CaseImportSkippedRow, actor importapp.Actor) {
	a.svc.RecordSkippedCaseImport(ctx, caseapp.CaseImportSkippedRow{
		RowIndex: row.RowIndex, CaseName: row.CaseName, Reasons: row.Reasons, RawValues: row.RawValues,
	}, actor.ActorID, actor.ActorRole, actor.IPAddress, actor.UserAgent)
}

// caregiverSiteLookup 讓 caregiver 匯入時以名稱比對據點。
type caregiverSiteLookup struct{ repo *masterinfra.SiteRepository }

func (a caregiverSiteLookup) GetByName(ctx context.Context, name string) (*caregiverapp.SiteRef, error) {
	s, err := a.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &caregiverapp.SiteRef{ID: s.ID, Name: s.Name}, nil
}

// caseDuplicateFinder 讓 caseimport 於 dry-run 階段透過 casemgmt 查重。
type caseDuplicateFinder struct{ svc *caseapp.CaseService }

func (a caseDuplicateFinder) FindDuplicate(ctx context.Context, nationalID, name string) (*importapp.DuplicateRef, error) {
	found, err := a.svc.FindPossibleDuplicate(ctx, nationalID, name)
	if err != nil || found == nil {
		return nil, err
	}
	return &importapp.DuplicateRef{CaseID: found.ID, CaseCode: found.Code, CaseName: found.Name}, nil
}
