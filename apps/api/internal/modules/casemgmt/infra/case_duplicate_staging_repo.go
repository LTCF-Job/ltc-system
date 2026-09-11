package infra

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// CaseDuplicateStagingRepository 提供 case_import_duplicate_rows 資料表之存取操作；
// 疑似重複個案在使用者裁決前只存在於這張暫存表，不會出現在 cases 表。
type CaseDuplicateStagingRepository struct {
	db *pgxpool.Pool
}

// NewCaseDuplicateStagingRepository 建立 CaseDuplicateStagingRepository 實例。
func NewCaseDuplicateStagingRepository(db *pgxpool.Pool) *CaseDuplicateStagingRepository {
	return &CaseDuplicateStagingRepository{db: db}
}

// Insert 寫入一筆暫存列；同一 (file_hash, row_key) 已存在時視為同一次匯入重試，
// 不重複寫入並回傳 alreadyStaged=true。
func (r *CaseDuplicateStagingRepository) Insert(ctx context.Context, cand app.DuplicateCandidate) (uuid.UUID, bool, error) {
	db := pgxdb.FromContext(ctx, r.db)
	var id uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO case_import_duplicate_rows (
			file_hash, row_key, row_index, sheet_name,
			name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
			household_type, gender, birth_date, birth_date_raw, national_id_invalid,
			care_contact_role, care_contact_name, registered_address, home_address,
			service_category, service_usage_type,
			site_id, site_name_raw, caregiver_id, remarks, duplicate_case_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		ON CONFLICT (file_hash, row_key) DO NOTHING
		RETURNING id
	`,
		cand.FileHash, cand.RowKey, cand.RowIndex, cand.SheetName,
		cand.Name, cand.NameNormalized, cand.NationalIDCipher, cand.NationalIDHMAC, cand.NationalIDMasked,
		cand.HouseholdType, cand.Gender, cand.BirthDate, cand.BirthDateRaw, cand.NationalIDInvalid,
		cand.CareContactRole, cand.CareContactName, cand.RegisteredAddress, cand.HomeAddress,
		cand.ServiceCategory, cand.ServiceUsageType,
		cand.SiteID, cand.SiteNameRaw, cand.CaregiverID, cand.Remarks, cand.DuplicateCaseID,
	).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, true, nil
		}
		return uuid.Nil, false, err
	}
	return id, false, nil
}

const duplicateStagingSelectColumns = `
	d.id, d.file_hash, d.row_key, d.row_index, d.sheet_name,
	d.name, d.name_normalized, d.national_id_cipher, d.national_id_hmac, d.national_id_masked,
	d.household_type, d.gender, d.birth_date, d.birth_date_raw, d.national_id_invalid,
	d.care_contact_role, d.care_contact_name, d.registered_address, d.home_address,
	d.service_category, d.service_usage_type,
	d.site_id, COALESCE(st.name, ''), d.site_name_raw,
	d.caregiver_id, COALESCE(cg.name, ''),
	d.remarks, d.duplicate_case_id, dc.name, d.status, d.resulting_case_id, d.resolved_at, d.resolved_by, d.created_at
`

const duplicateStagingFrom = `
	FROM case_import_duplicate_rows d
	JOIN cases dc ON dc.id = d.duplicate_case_id
	LEFT JOIN sites st ON st.id = d.site_id
	LEFT JOIN caregivers cg ON cg.id = d.caregiver_id
`

func scanDuplicateCandidate(row pgx.Row) (*app.DuplicateCandidate, error) {
	var c app.DuplicateCandidate
	err := row.Scan(
		&c.ID, &c.FileHash, &c.RowKey, &c.RowIndex, &c.SheetName,
		&c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked,
		&c.HouseholdType, &c.Gender, &c.BirthDate, &c.BirthDateRaw, &c.NationalIDInvalid,
		&c.CareContactRole, &c.CareContactName, &c.RegisteredAddress, &c.HomeAddress,
		&c.ServiceCategory, &c.ServiceUsageType,
		&c.SiteID, &c.SiteName, &c.SiteNameRaw,
		&c.CaregiverID, &c.CaregiverName,
		&c.Remarks, &c.DuplicateCaseID, &c.DuplicateCaseName, &c.Status, &c.ResultingCaseID, &c.ResolvedAt, &c.ResolvedBy, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListPending 取得所有待裁決（status=pending）的疑似重複個案暫存列。
func (r *CaseDuplicateStagingRepository) ListPending(ctx context.Context) ([]app.DuplicateCandidate, error) {
	db := pgxdb.FromContext(ctx, r.db)
	query := "SELECT " + duplicateStagingSelectColumns + duplicateStagingFrom + " WHERE d.status = 'pending' ORDER BY d.created_at ASC"
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending duplicate candidates: %w", err)
	}
	defer rows.Close()

	list := make([]app.DuplicateCandidate, 0)
	for rows.Next() {
		c, err := scanDuplicateCandidate(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pending duplicate candidates: %w", err)
	}
	return list, nil
}

// GetByID 依 ID 取得單筆暫存列；查無資料回傳 nil、nil。走 pgxdb.FromContext
// 讓 ResolveDuplicateCandidate 能在同一交易內讀到最新狀態。
func (r *CaseDuplicateStagingRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.DuplicateCandidate, error) {
	db := pgxdb.FromContext(ctx, r.db)
	query := "SELECT " + duplicateStagingSelectColumns + duplicateStagingFrom + " WHERE d.id = $1"
	c, err := scanDuplicateCandidate(db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

// Delete 移除尚未裁決的暫存列，供「忽略此筆」直接把資料從系統刪除；
// rowsAffected=0 代表該列不存在或已被裁決過（並發保護）。
func (r *CaseDuplicateStagingRepository) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
	db := pgxdb.FromContext(ctx, r.db)
	tag, err := db.Exec(ctx, `DELETE FROM case_import_duplicate_rows WHERE id = $1 AND status = 'pending'`, id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Resolve 將暫存列標記為裁決結果；rowsAffected=0 代表該列已被裁決過（並發保護）。
func (r *CaseDuplicateStagingRepository) Resolve(ctx context.Context, id uuid.UUID, status string, resolvedBy uuid.UUID, resultingCaseID *uuid.UUID) (int64, error) {
	db := pgxdb.FromContext(ctx, r.db)
	tag, err := db.Exec(ctx, `
		UPDATE case_import_duplicate_rows
		SET status = $2, resolved_at = now(), resolved_by = $3, resulting_case_id = $4
		WHERE id = $1 AND status = 'pending'
	`, id, status, resolvedBy, resultingCaseID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
