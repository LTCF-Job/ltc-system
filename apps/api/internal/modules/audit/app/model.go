package app

import (
	"time"

	"github.com/google/uuid"
)

// Entry 是一筆待寫入的稽核紀錄。BeforeData 與 AfterData 會被序列化進 audit_log
// 的 JSONB 欄位，其形狀由發動異動的能力自行決定並負責保持穩定。
type Entry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData interface{}
	AfterData  interface{}
	IPAddress  *string
	UserAgent  *string
}

// Record 是一筆已寫入的稽核紀錄。
type Record struct {
	ID         int64
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData interface{}
	AfterData  interface{}
	IPAddress  *string
	UserAgent  *string
	CreatedAt  time.Time
}

// Filter 是稽核紀錄的查詢條件，由 transport 依查詢參數組裝。
type Filter struct {
	ActorID    *uuid.UUID
	Action     string
	EntityType string
	EntityID   string
	StartDate  *time.Time
	EndDate    *time.Time
	Q          string
	Page       int
	PageSize   int
}
