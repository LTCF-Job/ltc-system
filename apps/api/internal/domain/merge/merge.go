package merge

import (
	"time"

	"github.com/google/uuid"
)

// RideSourceInput 代表單筆來自 Google 表單的回報來源紀錄。
type RideSourceInput struct {
	SourceID       uuid.UUID
	SourcePriority int
	VehicleID      uuid.UUID
	DriverID       *uuid.UUID
	Reported       string // "boarded" 或 "absent"
	SubmittedAt    time.Time
}

// ExistingRecordState 代表既有 ride_records 的狀態（用以保護人工裁決與人工更正）。
type ExistingRecordState struct {
	HasConflict        bool
	ConflictResolvedAt *time.Time
	ResolvedVehicleID  *uuid.UUID
	ResolvedDriverID   *uuid.UUID

	CorrectedAt      *time.Time
	CorrectedBy      *uuid.UUID
	EffectiveStatus  string // 既有更正後的狀態
	CorrectedVehicle *uuid.UUID
	CorrectedDriver  *uuid.UUID
}

// MergeResult 代表混車合併後的運算結果。
type MergeResult struct {
	MergedStatus    string // "boarded", "absent", "unreported"
	EffectiveStatus string // 最終生效狀態
	HasConflict     bool
	SelectedVehicle uuid.UUID
	SelectedDriver  *uuid.UUID
	SourceChanged   bool // 若人工更正後來源資料發生變更，設為 true
}

// MergeRideSources 依據規格書 4.6 執行「同車取最新、跨車 OR」之純合併演算法。
func MergeRideSources(
	sources []RideSourceInput,
	existing *ExistingRecordState,
	defaultVehicle uuid.UUID,
	defaultDriver *uuid.UUID,
) MergeResult {
	// 1. 依 vehicle_id 分組，每組取 submitted_at 最新的一筆為該車有效回報
	latestByVehicle := make(map[uuid.UUID]RideSourceInput)
	for _, src := range sources {
		current, exists := latestByVehicle[src.VehicleID]
		if !exists || sourcePreferred(src, current) {
			latestByVehicle[src.VehicleID] = src
		}
	}

	// 2. 統計各車有效回報
	var boardedSources []RideSourceInput
	totalValidReports := len(latestByVehicle)

	for _, src := range latestByVehicle {
		if src.Reported == "boarded" {
			boardedSources = append(boardedSources, src)
		}
	}

	// 3. 計算 mergedStatus
	mergedStatus := "unreported"
	if len(boardedSources) > 0 {
		mergedStatus = "boarded"
	} else if totalValidReports > 0 {
		mergedStatus = "absent"
	}

	// 4. 判斷是否有跨車衝突（兩台以上相異車輛皆回報有坐）
	hasConflict := len(boardedSources) > 1

	// 5. 決定承載車輛與司機
	var selectedVehicle uuid.UUID
	var selectedDriver *uuid.UUID

	if existing != nil && existing.ConflictResolvedAt != nil && existing.ResolvedVehicleID != nil {
		// 已人工裁決 -> 保留指定值
		selectedVehicle = *existing.ResolvedVehicleID
		selectedDriver = existing.ResolvedDriverID
	} else if existing != nil && existing.CorrectedAt != nil && existing.CorrectedVehicle != nil {
		// 已人工更正車輛 -> 保留
		selectedVehicle = *existing.CorrectedVehicle
		selectedDriver = existing.CorrectedDriver
	} else if len(boardedSources) > 0 {
		// 取最早回報有坐者之車輛與司機
		earliest := boardedSources[0]
		for _, b := range boardedSources[1:] {
			if b.SubmittedAt.Before(earliest.SubmittedAt) ||
				(b.SubmittedAt.Equal(earliest.SubmittedAt) && sourcePreferred(b, earliest)) {
				earliest = b
			}
		}
		selectedVehicle = earliest.VehicleID
		selectedDriver = earliest.DriverID
	} else {
		// 沒坐或未回報 -> 取排班預設值
		selectedVehicle = defaultVehicle
		selectedDriver = defaultDriver
	}

	// 6. 決定 effectiveStatus
	effectiveStatus := mergedStatus
	sourceChanged := false

	if existing != nil && existing.CorrectedAt != nil {
		// 已人工更正過：保持人工指定狀態，若司機來源有變更則提示 sourceChanged
		effectiveStatus = existing.EffectiveStatus
		if effectiveStatus != mergedStatus {
			sourceChanged = true
		}
	}

	return MergeResult{
		MergedStatus:    mergedStatus,
		EffectiveStatus: effectiveStatus,
		HasConflict:     hasConflict,
		SelectedVehicle: selectedVehicle,
		SelectedDriver:  selectedDriver,
		SourceChanged:   sourceChanged,
	}
}

// sourcePreferred 定義相同車輛來源的 deterministic tie-break：較新的提交時間優先，
// 再比較來源優先級，最後以 UUID 字串作穩定排序，避免 map iteration 影響申報結果。
func sourcePreferred(candidate, current RideSourceInput) bool {
	if candidate.SubmittedAt.After(current.SubmittedAt) {
		return true
	}
	if !candidate.SubmittedAt.Equal(current.SubmittedAt) {
		return false
	}
	if candidate.SourcePriority != current.SourcePriority {
		return candidate.SourcePriority > current.SourcePriority
	}
	return candidate.SourceID.String() > current.SourceID.String()
}
