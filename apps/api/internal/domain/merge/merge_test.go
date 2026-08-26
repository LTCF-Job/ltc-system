package merge

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMergeRideSources(t *testing.T) {
	vA := uuid.New()
	vB := uuid.New()
	vDefault := uuid.New()
	dA := uuid.New()
	dB := uuid.New()

	baseTime := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)

	t.Run("單車回報有坐", func(t *testing.T) {
		sources := []RideSourceInput{
			{VehicleID: vA, DriverID: &dA, Reported: "boarded", SubmittedAt: baseTime},
		}
		res := MergeRideSources(sources, nil, vDefault, nil)
		assert.Equal(t, "boarded", res.MergedStatus)
		assert.Equal(t, "boarded", res.EffectiveStatus)
		assert.False(t, res.HasConflict)
		assert.Equal(t, vA, res.SelectedVehicle)
		assert.Equal(t, &dA, res.SelectedDriver)
	})

	t.Run("單車回報沒坐", func(t *testing.T) {
		sources := []RideSourceInput{
			{VehicleID: vA, DriverID: &dA, Reported: "absent", SubmittedAt: baseTime},
		}
		res := MergeRideSources(sources, nil, vDefault, nil)
		assert.Equal(t, "absent", res.MergedStatus)
		assert.Equal(t, "absent", res.EffectiveStatus)
		assert.False(t, res.HasConflict)
		assert.Equal(t, vDefault, res.SelectedVehicle)
	})

	t.Run("無任何回報", func(t *testing.T) {
		res := MergeRideSources(nil, nil, vDefault, nil)
		assert.Equal(t, "unreported", res.MergedStatus)
		assert.Equal(t, "unreported", res.EffectiveStatus)
		assert.False(t, res.HasConflict)
		assert.Equal(t, vDefault, res.SelectedVehicle)
	})

	t.Run("A 車沒坐、B 車有坐 (跨車 OR)", func(t *testing.T) {
		sources := []RideSourceInput{
			{VehicleID: vA, DriverID: &dA, Reported: "absent", SubmittedAt: baseTime},
			{VehicleID: vB, DriverID: &dB, Reported: "boarded", SubmittedAt: baseTime.Add(time.Minute)},
		}
		res := MergeRideSources(sources, nil, vDefault, nil)
		assert.Equal(t, "boarded", res.MergedStatus)
		assert.Equal(t, "boarded", res.EffectiveStatus)
		assert.False(t, res.HasConflict)
		assert.Equal(t, vB, res.SelectedVehicle)
		assert.Equal(t, &dB, res.SelectedDriver)
	})

	t.Run("A 車有坐、B 車有坐 (衝突產生)", func(t *testing.T) {
		sources := []RideSourceInput{
			{VehicleID: vA, DriverID: &dA, Reported: "boarded", SubmittedAt: baseTime},
			{VehicleID: vB, DriverID: &dB, Reported: "boarded", SubmittedAt: baseTime.Add(time.Minute)},
		}
		res := MergeRideSources(sources, nil, vDefault, nil)
		assert.Equal(t, "boarded", res.MergedStatus)
		assert.True(t, res.HasConflict)
		assert.Equal(t, vA, res.SelectedVehicle) // 最早回報
	})

	t.Run("同車先報有坐、後報沒坐 (同車取最新)", func(t *testing.T) {
		sources := []RideSourceInput{
			{VehicleID: vA, DriverID: &dA, Reported: "boarded", SubmittedAt: baseTime},
			{VehicleID: vA, DriverID: &dA, Reported: "absent", SubmittedAt: baseTime.Add(10 * time.Minute)},
		}
		res := MergeRideSources(sources, nil, vDefault, nil)
		assert.Equal(t, "absent", res.MergedStatus)
		assert.False(t, res.HasConflict)
	})

	t.Run("已人工裁決後重跑同步 (車輛司機不被覆蓋)", func(t *testing.T) {
		sources := []RideSourceInput{
			{VehicleID: vA, DriverID: &dA, Reported: "boarded", SubmittedAt: baseTime},
			{VehicleID: vB, DriverID: &dB, Reported: "boarded", SubmittedAt: baseTime.Add(time.Minute)},
		}
		resolvedTime := baseTime.Add(2 * time.Hour)
		existing := &ExistingRecordState{
			HasConflict:        true,
			ConflictResolvedAt: &resolvedTime,
			ResolvedVehicleID:  &vB,
			ResolvedDriverID:   &dB,
		}
		res := MergeRideSources(sources, existing, vDefault, nil)
		assert.Equal(t, vB, res.SelectedVehicle) // 維持裁決之 B 車
		assert.Equal(t, &dB, res.SelectedDriver)
	})

	t.Run("已更正後來源改變 (effectiveStatus 不變且提示 sourceChanged)", func(t *testing.T) {
		// 司機原本未回報或報沒坐，人工更正為 boarded
		correctedTime := baseTime.Add(time.Hour)
		operatorID := uuid.New()
		existing := &ExistingRecordState{
			CorrectedAt:     &correctedTime,
			CorrectedBy:     &operatorID,
			EffectiveStatus: "boarded",
		}

		// 事後司機同步時回報了 absent
		sources := []RideSourceInput{
			{VehicleID: vA, DriverID: &dA, Reported: "absent", SubmittedAt: baseTime},
		}

		res := MergeRideSources(sources, existing, vDefault, nil)
		assert.Equal(t, "absent", res.MergedStatus)
		assert.Equal(t, "boarded", res.EffectiveStatus) // 維持人工更正的 boarded
		assert.True(t, res.SourceChanged)               // 標記來源資料有變更
	})
}
