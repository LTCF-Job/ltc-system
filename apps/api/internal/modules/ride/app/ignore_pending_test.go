package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ingestOneBoardedLeg 匯入一筆「有坐」的去程，回傳個案、服務日期與該次提交的 ID，
// 讓忽略路徑的測試從一個真的有搭乘來源與搭乘紀錄的狀態出發。
func ingestOneBoardedLeg(t *testing.T, store *fakeRecordStore, caseID uuid.UUID) (time.Time, uuid.UUID) {
	t.Helper()

	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	driverID := uuid.New()
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	result, err := svc.IngestSubmission(context.Background(), uuid.New(), uuid.New(), ProcessSubmissionRequest{
		ServiceDate: serviceDate,
		DriverID:    &driverID,
		Answers:     map[string]string{"1.吳桂 [去程]": "有坐"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Written)
	require.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 1}], "前置條件：應已產生搭乘紀錄")

	return serviceDate, store.submission
}

func TestRideService_DeleteSubmission_RemovesSourcesAndRecalculatesRecord(t *testing.T) {
	caseID := uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	_, submissionID := ingestOneBoardedLeg(t, store, caseID)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	rowsAffected, err := svc.DeleteSubmission(context.Background(), submissionID)

	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
	assert.NotContains(t, store.submissions, submissionID)
	assert.Empty(t, store.sources, "提交紀錄被刪除時，其展開出的搭乘來源要一併消失")
	assert.Empty(t, store.records,
		"搭乘來源全數消失後必須重算掉衍生的搭乘紀錄，否則月曆會留下對不上任何來源的資料")
}

func TestRideService_DeleteSubmission_NotFoundIsNoop(t *testing.T) {
	caseID := uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	ingestOneBoardedLeg(t, store, caseID)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	rowsAffected, err := svc.DeleteSubmission(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Zero(t, rowsAffected, "查無提交紀錄回 0 列，由呼叫端翻成待維護項目不存在")
	assert.Len(t, store.records, 1, "沒刪到東西就不得動到既有搭乘紀錄")
}

func TestRideService_DeleteRowConflict_RemovesConflictOnly(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	conflictID := uuid.New()

	store := newFakeRecordStore(nil)
	store.rowConflicts[conflictID] = &fakeRowConflict{id: conflictID, input: RowConflictInput{
		CaseID: caseID, ServiceDate: serviceDate, LegSeq: 1, VehicleID: vehicleID,
	}}
	store.openRowConflicts[rowConflictSlotKey{vehicleID, caseID, "2026-03-02", 1}] = conflictID
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	rowsAffected, err := svc.DeleteRowConflict(context.Background(), conflictID)

	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
	assert.NotContains(t, store.rowConflicts, conflictID)
	assert.Empty(t, store.sources, "忽略衝突不得寫入暫存的新值")
	assert.Empty(t, store.records, "忽略衝突不得重算搭乘紀錄，既有資料維持衝突前的值")
}

func TestRideService_DeleteRowConflict_AlreadyGoneReturnsZero(t *testing.T) {
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	rowsAffected, err := svc.DeleteRowConflict(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Zero(t, rowsAffected, "衝突已被他人裁決或不存在時回 0 列，不視為錯誤")
}
