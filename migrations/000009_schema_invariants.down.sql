ALTER TABLE trainings DROP CONSTRAINT IF EXISTS chk_scheduled_training_has_subscription;
ALTER TABLE trainings DROP CONSTRAINT IF EXISTS chk_trainings_runtime_status;

ALTER TABLE trainings DROP CONSTRAINT IF EXISTS trainings_subscription_client_fkey;
ALTER TABLE trainings
    ADD CONSTRAINT trainings_subscription_id_fkey
    FOREIGN KEY (subscription_id) REFERENCES client_subscriptions(id) ON DELETE RESTRICT;
ALTER TABLE client_subscriptions DROP CONSTRAINT IF EXISTS uq_client_subscriptions_id_client;

ALTER TABLE training_requests DROP CONSTRAINT IF EXISTS training_requests_client_id_fkey;
ALTER TABLE training_requests
    ADD CONSTRAINT training_requests_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE payments ALTER COLUMN client_id DROP NOT NULL;
ALTER TABLE client_subscriptions ALTER COLUMN product_id DROP NOT NULL;
ALTER TABLE client_subscriptions ALTER COLUMN client_id DROP NOT NULL;
ALTER TABLE trainings ALTER COLUMN client_id DROP NOT NULL;
ALTER TABLE training_requests ALTER COLUMN client_id DROP NOT NULL;

-- Rows converted from legacy 'pending' trainings are intentionally not converted back:
-- there is no reliable way to distinguish them from trainings that were already scheduled.
