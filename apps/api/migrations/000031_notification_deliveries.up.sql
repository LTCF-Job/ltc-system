-- 以通知事件與收件人為粒度保存派送狀態，避免部分成功 retry 重複寄送。
CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id TEXT NOT NULL,
    recipient_key TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('processing', 'sent', 'failed')),
    provider_message_id TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_notification_delivery_event_recipient UNIQUE (event_id, recipient_key)
);

CREATE INDEX idx_notification_deliveries_retry
    ON notification_deliveries (status, claimed_at);
