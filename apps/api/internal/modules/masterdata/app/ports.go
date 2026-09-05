package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SiteStore 定義單位主檔的讀寫邊界。
type SiteStore interface {
	List(ctx context.Context, region, q, status string, page, pageSize int) ([]Site, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Site, error)
	Create(ctx context.Context, s *Site) error
	Update(ctx context.Context, s *Site) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// VehicleStore 定義車輛主檔的讀寫邊界。
type VehicleStore interface {
	List(ctx context.Context, filter VehicleFilter, page, pageSize int) ([]Vehicle, int64, error)
	Create(ctx context.Context, v *Vehicle) error
	Update(ctx context.Context, v *Vehicle) error
	SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error)
	CountActiveDriverAssignments(ctx context.Context, vehicleID uuid.UUID) (int, error)
	CountScheduleLegs(ctx context.Context, vehicleID uuid.UUID) (int, error)
}

// DriverStore 定義司機主檔與車輛指派的讀寫邊界。
type DriverStore interface {
	List(ctx context.Context, region, q, status string, page, pageSize int) ([]Driver, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Driver, error)
	Create(ctx context.Context, d *Driver) error
	Update(ctx context.Context, d *Driver) error
	AssignVehicle(ctx context.Context, a *DriverAssignment) error
	ListByVehicleIDsOnDate(ctx context.Context, vehicleIDs []uuid.UUID, on time.Time) (map[uuid.UUID][]Driver, error)
	ReplaceVehicleDrivers(ctx context.Context, vehicleID uuid.UUID, driverIDs []uuid.UUID, effectiveFrom time.Time) error
	SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error)
	CloseActiveAssignments(ctx context.Context, driverID uuid.UUID) error
}

// TransactionRunner 讓涉及司機主檔與車輛指派的多步驟異動共用同一筆交易。
type TransactionRunner interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// RegionStore 定義區域主檔的讀寫邊界。
type RegionStore interface {
	List(ctx context.Context, q, status string, page, pageSize int) ([]Region, int64, error)
	ListAll(ctx context.Context) ([]Region, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Region, error)
	GetByName(ctx context.Context, name string) (*Region, error)
	Create(ctx context.Context, r *Region) error
	Update(ctx context.Context, r *Region) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AuditWriter 定義主檔異動留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}
