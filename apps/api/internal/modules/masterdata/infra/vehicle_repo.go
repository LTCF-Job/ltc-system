package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

// vehicleRow 是 vehicles 資料表的一列，另帶所屬單位 join 出來的名稱與區域。
type vehicleRow struct {
	ID                        uuid.UUID
	PlateNo                   string
	DisplayName               string
	SiteID                    *uuid.UUID
	SiteName                  *string
	SiteRegion                *string
	OwnerName                 *string
	Brand                     *string
	Model                     *string
	ManufactureYM             *string
	CompulsoryInsuranceExpiry *time.Time
	PassengerInsuranceExpiry  *time.Time
	ThirdPartyInsuranceExpiry *time.Time
	LastInspectionDate        *time.Time
	WheelchairAccessible      *bool
	Status                    string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (r vehicleRow) toApp() app.Vehicle {
	return app.Vehicle{
		ID:                        r.ID,
		PlateNo:                   r.PlateNo,
		DisplayName:               r.DisplayName,
		SiteID:                    r.SiteID,
		SiteName:                  derefString(r.SiteName),
		Region:                    derefString(r.SiteRegion),
		OwnerName:                 derefString(r.OwnerName),
		Brand:                     derefString(r.Brand),
		Model:                     derefString(r.Model),
		ManufactureYM:             derefString(r.ManufactureYM),
		CompulsoryInsuranceExpiry: r.CompulsoryInsuranceExpiry,
		PassengerInsuranceExpiry:  r.PassengerInsuranceExpiry,
		ThirdPartyInsuranceExpiry: r.ThirdPartyInsuranceExpiry,
		LastInspectionDate:        r.LastInspectionDate,
		WheelchairAccessible:      r.WheelchairAccessible,
		Status:                    r.Status,
		CreatedAt:                 r.CreatedAt,
		UpdatedAt:                 r.UpdatedAt,
	}
}

const vehicleSelect = `
	SELECT v.id, v.plate_no, v.display_name, v.site_id, s.name, s.region,
	       v.owner_name, v.brand, v.model, v.manufacture_ym,
	       v.compulsory_insurance_expiry, v.passenger_insurance_expiry,
	       v.third_party_insurance_expiry, v.last_inspection_date,
	       v.wheelchair_accessible, v.status, v.created_at, v.updated_at
	FROM vehicles v
	LEFT JOIN sites s ON s.id = v.site_id
`

func scanVehicle(dest *vehicleRow) []interface{} {
	return []interface{}{
		&dest.ID, &dest.PlateNo, &dest.DisplayName, &dest.SiteID, &dest.SiteName, &dest.SiteRegion,
		&dest.OwnerName, &dest.Brand, &dest.Model, &dest.ManufactureYM,
		&dest.CompulsoryInsuranceExpiry, &dest.PassengerInsuranceExpiry,
		&dest.ThirdPartyInsuranceExpiry, &dest.LastInspectionDate,
		&dest.WheelchairAccessible, &dest.Status, &dest.CreatedAt, &dest.UpdatedAt,
	}
}

// VehicleRepository 提供 vehicles 資料表之存取操作。
type VehicleRepository struct {
	db *pgxpool.Pool
}

// NewVehicleRepository 建立 VehicleRepository 實例。
func NewVehicleRepository(db *pgxpool.Pool) *VehicleRepository {
	return &VehicleRepository{db: db}
}

// vehicleFilterSQL 是 List 與其 count 查詢共用的條件；區域條件走所屬單位，車輛本身不存區域。
const vehicleFilterSQL = `
	WHERE ($1::uuid IS NULL OR v.site_id = $1)
	  AND ($2 = '' OR s.region = $2)
	  AND ($3 = '' OR v.plate_no ILIKE '%' || $3 || '%' OR v.display_name ILIKE '%' || $3 || '%')
`

// List 取得車輛清單。
func (r *VehicleRepository) List(ctx context.Context, filter app.VehicleFilter, page, pageSize int) ([]app.Vehicle, int64, error) {
	if r.db == nil {
		return []app.Vehicle{}, 0, nil
	}
	offset := (page - 1) * pageSize
	query := vehicleSelect + vehicleFilterSQL + `
		ORDER BY v.display_name ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, query, filter.SiteID, filter.Region, filter.Q, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query vehicles: %w", err)
	}
	defer rows.Close()

	var list []app.Vehicle
	for rows.Next() {
		var v vehicleRow
		if err := rows.Scan(scanVehicle(&v)...); err != nil {
			return nil, 0, err
		}
		list = append(list, v.toApp())
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM vehicles v
		LEFT JOIN sites s ON s.id = v.site_id
	` + vehicleFilterSQL
	_ = r.db.QueryRow(ctx, countQuery, filter.SiteID, filter.Region, filter.Q).Scan(&total)

	return list, total, nil
}

// GetByID 依 UUID 取得車輛。
func (r *VehicleRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Vehicle, error) {
	return r.getOne(ctx, vehicleSelect+` WHERE v.id = $1`, id)
}

// GetByDisplayName 依顯示名稱尋找（支援匯入比對）。
func (r *VehicleRepository) GetByDisplayName(ctx context.Context, displayName string) (*app.Vehicle, error) {
	return r.getOne(ctx, vehicleSelect+` WHERE v.display_name = $1 LIMIT 1`, displayName)
}

func (r *VehicleRepository) getOne(ctx context.Context, query string, arg interface{}) (*app.Vehicle, error) {
	var v vehicleRow
	if err := r.db.QueryRow(ctx, query, arg).Scan(scanVehicle(&v)...); err != nil {
		return nil, err
	}
	vehicle := v.toApp()
	return &vehicle, nil
}

func vehicleWriteArgs(v *app.Vehicle) []interface{} {
	return []interface{}{
		v.ID, v.PlateNo, v.DisplayName, v.SiteID,
		nullableText(v.OwnerName), nullableText(v.Brand), nullableText(v.Model), nullableText(v.ManufactureYM),
		v.CompulsoryInsuranceExpiry, v.PassengerInsuranceExpiry, v.ThirdPartyInsuranceExpiry,
		v.LastInspectionDate, v.WheelchairAccessible, v.Status,
	}
}

// nullableText 讓未填寫的選填欄位寫入 NULL 而不是空字串，避免同一語意出現兩種表示。
func nullableText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Create 新增車輛。
func (r *VehicleRepository) Create(ctx context.Context, v *app.Vehicle) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	query := `
		INSERT INTO vehicles (
			id, plate_no, display_name, site_id, owner_name, brand, model, manufacture_ym,
			compulsory_insurance_expiry, passenger_insurance_expiry, third_party_insurance_expiry,
			last_inspection_date, wheelchair_accessible, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at
	`
	if err := r.db.QueryRow(ctx, query, vehicleWriteArgs(v)...).Scan(&v.CreatedAt, &v.UpdatedAt); err != nil {
		return err
	}
	return r.fillSite(ctx, v)
}

// Update 修改車輛。
func (r *VehicleRepository) Update(ctx context.Context, v *app.Vehicle) error {
	query := `
		UPDATE vehicles
		SET plate_no = $2, display_name = $3, site_id = $4, owner_name = $5, brand = $6,
		    model = $7, manufacture_ym = $8, compulsory_insurance_expiry = $9,
		    passenger_insurance_expiry = $10, third_party_insurance_expiry = $11,
		    last_inspection_date = $12, wheelchair_accessible = $13, status = $14,
		    updated_at = now()
		WHERE id = $1
		RETURNING created_at, updated_at
	`
	if err := r.db.QueryRow(ctx, query, vehicleWriteArgs(v)...).Scan(&v.CreatedAt, &v.UpdatedAt); err != nil {
		return err
	}
	return r.fillSite(ctx, v)
}

// fillSite 補上寫入結果的所屬單位名稱與區域，讓回應與 List 的形狀一致。
func (r *VehicleRepository) fillSite(ctx context.Context, v *app.Vehicle) error {
	v.SiteName, v.Region = "", ""
	if v.SiteID == nil {
		return nil
	}
	var name, region string
	err := r.db.QueryRow(ctx, `SELECT name, region FROM sites WHERE id = $1`, *v.SiteID).Scan(&name, &region)
	if err != nil {
		return err
	}
	v.SiteName, v.Region = name, region
	return nil
}
