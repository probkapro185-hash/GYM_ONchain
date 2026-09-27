DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS staff_tasks;
DROP TABLE IF EXISTS client_notes;
DROP TABLE IF EXISTS client_progress;
ALTER TABLE client_subscriptions DROP COLUMN IF EXISTS freeze_days;
ALTER TABLE client_subscriptions DROP COLUMN IF EXISTS frozen_at;
