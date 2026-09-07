package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type fuelSnapshotReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*FuelLog, error)
}

type maintenanceSnapshotReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*MaintenanceLog, error)
}

func loadFuelAuditSnapshot(ctx context.Context, store FuelStore, id uuid.UUID) (interface{}, error) {
	reader, ok := store.(fuelSnapshotReader)
	if !ok {
		return nil, nil
	}
	item, err := reader.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("fuel log not found")
	}
	return item.AuditSnapshot(), nil
}

func loadMaintenanceAuditSnapshot(ctx context.Context, store MaintenanceStore, id uuid.UUID) (interface{}, error) {
	reader, ok := store.(maintenanceSnapshotReader)
	if !ok {
		return nil, nil
	}
	item, err := reader.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("maintenance log not found")
	}
	return item.AuditSnapshot(), nil
}

func auditContextOrEmpty(contexts []AuditContext) AuditContext {
	if len(contexts) == 0 {
		return AuditContext{}
	}
	return contexts[0]
}

// writeAuditBestEffort 將已完成的非外部營運 mutation 記錄至 audit；錯誤不可被
// 靜默吞掉，也不應把已完成的資料異動回報成失敗。
func writeAuditBestEffort(ctx context.Context, writer AuditWriter, actorID *uuid.UUID, actorRole *string, source AuditContext, action, entityType string, entityID uuid.UUID, before, after interface{}) {
	if writer == nil {
		return
	}
	id := entityID.String()
	if err := writer.Write(ctx, AuditEntry{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     action,
		EntityType: entityType,
		EntityID:   &id,
		BeforeData: before,
		AfterData:  after,
		IPAddress:  source.IPAddress,
		UserAgent:  source.UserAgent,
	}); err != nil {
		slog.Error("ops audit write failed", "action", action, "entity_type", entityType, "entity_id", id, "error", err)
	}
}
