-- Migration: 000002_create_notification_log.up.sql
-- Description: 建立通知歷史日誌資料表

CREATE TABLE IF NOT EXISTS notification_log (
    id BIGSERIAL PRIMARY KEY,
    topic TEXT NOT NULL CHECK (topic IN ('missing_report', 'driver_leave', 'month_end', 'export_failed')),
    channel TEXT NOT NULL DEFAULT 'email' CHECK (channel IN ('email', 'sms', 'system')),
    recipient_emails TEXT[] NOT NULL DEFAULT '{}',
    subject TEXT NOT NULL,
    content_summary TEXT,
    payload JSONB,
    status TEXT NOT NULL CHECK (status IN ('sent', 'failed')),
    error_message TEXT,
    triggered_by UUID,
    triggered_by_name TEXT,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notification_log_topic ON notification_log(topic);
CREATE INDEX IF NOT EXISTS idx_notification_log_sent_at ON notification_log(sent_at);
