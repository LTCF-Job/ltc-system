package main

import (
	"context"

	auditapp "ltc-system/apps/api/internal/modules/audit/app"
	caseapp "ltc-system/apps/api/internal/modules/casemgmt/app"
	drapp "ltc-system/apps/api/internal/modules/driverreport/app"
	holidayapp "ltc-system/apps/api/internal/modules/holiday/app"
	identityapp "ltc-system/apps/api/internal/modules/identity/app"
	masterapp "ltc-system/apps/api/internal/modules/masterdata/app"
	notifyapp "ltc-system/apps/api/internal/modules/notification/app"
	opsapp "ltc-system/apps/api/internal/modules/ops/app"
	reportingapp "ltc-system/apps/api/internal/modules/reporting/app"
	rideapp "ltc-system/apps/api/internal/modules/ride/app"
)

// 稽核寫入是跨能力的共用行為：audit 模組獨佔 audit_log 的 SQL，其他模組各自宣告
// 自己的 AuditWriter port。以下 adapter 只存在於 composition root，讓任何一個模組
// 都不需要直接依賴 audit 模組的型別。

type caseAuditWriter struct{ svc *auditapp.Service }

func (w caseAuditWriter) Write(ctx context.Context, e caseapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, BeforeData: e.BeforeData, AfterData: e.AfterData,
		IPAddress: e.IPAddress, UserAgent: e.UserAgent,
	})
}

type identityAuditWriter struct{ svc *auditapp.Service }

func (w identityAuditWriter) Write(ctx context.Context, e identityapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, BeforeData: e.BeforeData, AfterData: e.AfterData,
	})
}

type masterdataAuditWriter struct{ svc *auditapp.Service }

func (w masterdataAuditWriter) Write(ctx context.Context, e masterapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, BeforeData: e.BeforeData, AfterData: e.AfterData,
		IPAddress: e.IPAddress, UserAgent: e.UserAgent,
	})
}

type rideAuditWriter struct{ svc *auditapp.Service }

func (w rideAuditWriter) Write(ctx context.Context, e rideapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, BeforeData: e.BeforeData, AfterData: e.AfterData,
		IPAddress: e.IPAddress, UserAgent: e.UserAgent,
	})
}

type driverReportAuditWriter struct{ svc *auditapp.Service }

func (w driverReportAuditWriter) Write(ctx context.Context, e drapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, AfterData: e.AfterData,
		IPAddress: e.IPAddress, UserAgent: e.UserAgent,
	})
}

type holidayAuditWriter struct{ svc *auditapp.Service }

func (w holidayAuditWriter) Write(ctx context.Context, e holidayapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action,
		EntityType: e.EntityType, EntityID: e.EntityID, AfterData: e.AfterData,
	})
}

type notificationAuditWriter struct{ svc *auditapp.Service }

func (w notificationAuditWriter) Write(ctx context.Context, e notifyapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, BeforeData: e.BeforeData, AfterData: e.AfterData,
	})
}

type opsAuditWriter struct{ svc *auditapp.Service }

func (w opsAuditWriter) Write(ctx context.Context, e opsapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, BeforeData: e.BeforeData, AfterData: e.AfterData,
	})
}

type reportingAuditWriter struct{ svc *auditapp.Service }

func (w reportingAuditWriter) Write(ctx context.Context, e reportingapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, AfterData: e.AfterData,
	})
}
