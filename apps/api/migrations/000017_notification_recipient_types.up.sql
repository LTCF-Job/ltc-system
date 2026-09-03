ALTER TABLE notification_recipients
  ADD COLUMN recipient_type TEXT NOT NULL DEFAULT 'email'
    CHECK (recipient_type IN ('email', 'role', 'user')),
  ADD COLUMN target_role TEXT,
  ADD COLUMN user_id UUID,
  ALTER COLUMN email DROP NOT NULL;

-- 每種型別各自的識別欄位必須恰好填一個
ALTER TABLE notification_recipients ADD CONSTRAINT chk_recipient_target CHECK (
  (recipient_type = 'email' AND email IS NOT NULL AND target_role IS NULL AND user_id IS NULL) OR
  (recipient_type = 'role'  AND target_role IS NOT NULL AND email IS NULL AND user_id IS NULL) OR
  (recipient_type = 'user'  AND user_id IS NOT NULL AND email IS NULL AND target_role IS NULL)
);

ALTER TABLE notification_recipients DROP CONSTRAINT IF EXISTS uq_notification_topic_email;
CREATE UNIQUE INDEX uq_notification_recipients_email ON notification_recipients (topic, email) WHERE recipient_type = 'email';
CREATE UNIQUE INDEX uq_notification_recipients_role ON notification_recipients (topic, target_role) WHERE recipient_type = 'role';
CREATE UNIQUE INDEX uq_notification_recipients_user ON notification_recipients (topic, user_id) WHERE recipient_type = 'user';
