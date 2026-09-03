-- 反向操作前需先清除非 email 型別的列，否則 NOT NULL 還原會失敗。
DELETE FROM notification_recipients WHERE recipient_type != 'email';

DROP INDEX IF EXISTS uq_notification_recipients_email;
DROP INDEX IF EXISTS uq_notification_recipients_role;
DROP INDEX IF EXISTS uq_notification_recipients_user;

ALTER TABLE notification_recipients DROP CONSTRAINT IF EXISTS chk_recipient_target;
ALTER TABLE notification_recipients ALTER COLUMN email SET NOT NULL;
ALTER TABLE notification_recipients DROP COLUMN recipient_type;
ALTER TABLE notification_recipients DROP COLUMN target_role;
ALTER TABLE notification_recipients DROP COLUMN user_id;
ALTER TABLE notification_recipients ADD CONSTRAINT uq_notification_topic_email UNIQUE (topic, email);
