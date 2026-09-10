//go:build integration

package infra

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestRideRepository_ListRideRecordsInRange_KeywordFilter 鎖住 keyword 篩選的參數位移
// 回歸：ListRideRecordsInRange 曾用 $4 綁 keyword 但只傳 3 個參數，pgx 會直接回
// insufficient arguments 500 錯誤（同一類問題也出現在 ListCalendarCases，見
// eb2063b 的修正）。
func TestRideRepository_ListRideRecordsInRange_KeywordFilter(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/ltc_system?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("real Postgres not reachable at %s: %v", dbURL, err)
	}

	ctx := context.Background()
	suffix := time.Now().UnixNano()

	var vehicleID string
	err = pool.QueryRow(ctx, `
		INSERT INTO vehicles (plate_no, display_name)
		VALUES ($1, $2)
		RETURNING id
	`, "TEST-PLATE-"+timeSuffix(suffix), "測試車輛-"+timeSuffix(suffix)).Scan(&vehicleID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM vehicles WHERE id = $1`, vehicleID)
	})

	caseName := "關鍵字測試個案-" + timeSuffix(suffix)
	var caseID string
	err = pool.QueryRow(ctx, `
		INSERT INTO cases (name, name_normalized, site_name_raw)
		VALUES ($1, $1, 'test-site')
		RETURNING id
	`, caseName).Scan(&caseID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM cases WHERE id = $1`, caseID)
	})

	serviceDate := time.Now().Truncate(24 * time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO ride_records (case_id, service_date, leg_seq, merged_status, effective_status, vehicle_id)
		VALUES ($1, $2, 1, 'boarded', 'boarded', $3)
	`, caseID, serviceDate, vehicleID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM ride_records WHERE case_id = $1`, caseID)
	})

	repo := NewRideRepository(pool)
	start := serviceDate.AddDate(0, 0, -1)
	end := serviceDate.AddDate(0, 0, 1)

	records, err := repo.ListRideRecordsInRange(ctx, start, end, caseName)
	require.NoError(t, err, "keyword 篩選不應觸發 pgx insufficient arguments 錯誤")
	require.Len(t, records, 1)
	require.Equal(t, caseID, records[0].CaseID.String())

	noMatch, err := repo.ListRideRecordsInRange(ctx, start, end, "不存在的關鍵字-"+timeSuffix(suffix))
	require.NoError(t, err)
	require.Empty(t, noMatch)
}

func timeSuffix(n int64) string {
	return time.Unix(0, n).Format("150405.000000000")
}
