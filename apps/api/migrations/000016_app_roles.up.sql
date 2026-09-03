CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    tag_type TEXT NOT NULL DEFAULT 'info' CHECK (tag_type IN ('primary', 'success', 'warning', 'danger', 'info')),
    is_system BOOLEAN NOT NULL DEFAULT false,
    -- auth.RequireRoles 只認得 viewer/staff/admin；自訂角色必須對映到其中之一才能通過 API 層權限檢查
    base_role TEXT NOT NULL DEFAULT 'viewer' CHECK (base_role IN ('viewer', 'staff', 'admin')),
    permissions JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO roles (key, name, description, tag_type, is_system, base_role, permissions) VALUES
('admin', '系統管理員', '完整系統存取權限，可管理使用者、角色與所有功能模組', 'danger', true, 'admin', '{
  "dashboard": {"view": true, "edit": true}, "masters_regions": {"view": true, "edit": true},
  "masters_cases": {"view": true, "edit": true}, "masters_sites": {"view": true, "edit": true},
  "masters_vehicles": {"view": true, "edit": true}, "masters_drivers": {"view": true, "edit": true},
  "masters_caregivers": {"view": true, "edit": true}, "driver_reports": {"view": true, "edit": true},
  "driver_report_mappings": {"view": true, "edit": true}, "rides_calendar": {"view": true, "edit": true},
  "rides_issues": {"view": true, "edit": true}, "rides_missing": {"view": true, "edit": true},
  "reports_trip_summary": {"view": true, "edit": true}, "reports_hsinchu_schedule": {"view": true, "edit": true},
  "vehicles_maintenance": {"view": true, "edit": true}, "attendance_fuel": {"view": true, "edit": true},
  "exports": {"view": true, "edit": true}, "audit_logs": {"view": true, "edit": true},
  "settings_notifications": {"view": true, "edit": true}, "settings_users": {"view": true, "edit": true},
  "settings_roles": {"view": true, "edit": true}
}'::jsonb),
('viewer', '檢視人員', '僅能檢視系統資料，無法進行任何異動', 'info', true, 'viewer', '{
  "dashboard": {"view": true, "edit": false}, "masters_regions": {"view": true, "edit": false},
  "masters_cases": {"view": true, "edit": false}, "masters_sites": {"view": true, "edit": false},
  "masters_vehicles": {"view": true, "edit": false}, "masters_drivers": {"view": true, "edit": false},
  "masters_caregivers": {"view": true, "edit": false}, "driver_reports": {"view": true, "edit": false},
  "driver_report_mappings": {"view": true, "edit": false}, "rides_calendar": {"view": true, "edit": false},
  "rides_issues": {"view": true, "edit": false}, "rides_missing": {"view": true, "edit": false},
  "reports_trip_summary": {"view": true, "edit": false}, "reports_hsinchu_schedule": {"view": true, "edit": false},
  "vehicles_maintenance": {"view": true, "edit": false}, "attendance_fuel": {"view": true, "edit": false},
  "exports": {"view": true, "edit": false}, "audit_logs": {"view": false, "edit": false},
  "settings_notifications": {"view": true, "edit": false}, "settings_users": {"view": false, "edit": false},
  "settings_roles": {"view": false, "edit": false}
}'::jsonb)
ON CONFLICT (key) DO NOTHING;
