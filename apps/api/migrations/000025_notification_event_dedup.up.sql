-- Notification scheduler 的事件去重狀態；processing 失敗時由 application release，允許安全重試。
CREATE TABLE notification_event_dedup (
    dedup_key TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('processing', 'sent')),
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ
);

CREATE INDEX idx_notification_event_dedup_status ON notification_event_dedup(status, claimed_at);
