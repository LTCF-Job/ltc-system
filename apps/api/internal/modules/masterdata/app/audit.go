package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// writeAuditBestEffort 將已完成的主檔異動送入稽核；稽核服務故障不得讓已完成的
// 非外部 mutation 被回報成失敗，但也不可以靜默吞掉錯誤。
func writeAuditBestEffort(ctx context.Context, writer AuditWriter, actor ActorContext, action, entityType string, entityID uuid.UUID, before, after interface{}) {
	if writer == nil {
		return
	}
	id := entityID.String()
	if err := writer.Write(ctx, AuditEntry{
		ActorID:    &actor.ActorID,
		ActorRole:  &actor.ActorRole,
		Action:     action,
		EntityType: entityType,
		EntityID:   &id,
		BeforeData: before,
		AfterData:  after,
		IPAddress:  &actor.IPAddress,
		UserAgent:  &actor.UserAgent,
	}); err != nil {
		slog.Error("masterdata audit write failed", "action", action, "entity_type", entityType, "entity_id", id, "error", err)
	}
}

func actorOrEmpty(actors []ActorContext) ActorContext {
	if len(actors) == 0 {
		return ActorContext{}
	}
	return actors[0]
}

// VehicleSnapshotReader 是可選的車輛讀取能力。VehicleStore 保持最小寫入契約，
// 需要稽核 before 快照時才由實際 repository 提供這個額外介面。
type VehicleSnapshotReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Vehicle, error)
}

func loadVehicleAuditSnapshot(ctx context.Context, store VehicleStore, id uuid.UUID) (interface{}, error) {
	reader, ok := store.(VehicleSnapshotReader)
	if !ok {
		return nil, nil
	}
	vehicle, err := reader.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrVehicleNotFound) {
			return nil, ErrVehicleNotFound
		}
		return nil, fmt.Errorf("failed to load vehicle audit snapshot: %w", err)
	}
	if vehicle == nil {
		return nil, ErrVehicleNotFound
	}
	snapshot := vehicle.AuditSnapshot()
	return snapshot, nil
}
