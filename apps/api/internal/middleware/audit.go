package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditLogEntry 代表欲寫入 audit_log 之結構體。
type AuditLogEntry struct {
	ActorID    uuid.UUID
	ActorRole  string
	Action     string // create, update, delete, reveal_pii, correct, resolve_conflict, export, setting_change, import
	EntityType string
	EntityID   string
	BeforeData interface{}
	AfterData  interface{}
	IPAddress  string
	UserAgent  string
}

// RecordAuditLog 將操作稽核日誌非同步或同步寫入 PostgreSQL 之 audit_log 資料表。
func RecordAuditLog(ctx context.Context, db *pgxpool.Pool, entry AuditLogEntry) error {
	var beforeJSON, afterJSON []byte
	var err error

	if entry.BeforeData != nil {
		beforeJSON, err = json.Marshal(entry.BeforeData)
		if err != nil {
			slog.Error("Failed to marshal before data for audit log", slog.String("error", err.Error()))
		}
	}
	if entry.AfterData != nil {
		afterJSON, err = json.Marshal(entry.AfterData)
		if err != nil {
			slog.Error("Failed to marshal after data for audit log", slog.String("error", err.Error()))
		}
	}

	query := `
		INSERT INTO audit_log (
			actor_id, actor_role, action, entity_type, entity_id,
			before_data, after_data, ip_address, user_agent
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	var actorIDVal *uuid.UUID
	if entry.ActorID != uuid.Nil {
		actorIDVal = &entry.ActorID
	}

	_, err = db.Exec(ctx, query,
		actorIDVal,
		entry.ActorRole,
		entry.Action,
		entry.EntityType,
		entry.EntityID,
		beforeJSON,
		afterJSON,
		entry.IPAddress,
		entry.UserAgent,
	)

	if err != nil {
		slog.Error("Failed to insert audit log",
			slog.String("action", entry.Action),
			slog.String("entity", entry.EntityType),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to record audit log: %w", err)
	}

	return nil
}
