DROP INDEX IF EXISTS idx_trainings_subscription;
ALTER TABLE trainings DROP CONSTRAINT IF EXISTS trainings_subscription_id_fkey;
ALTER TABLE trainings DROP COLUMN IF EXISTS subscription_id;
