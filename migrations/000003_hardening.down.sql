ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_product_shape;
ALTER TABLE client_subscriptions DROP CONSTRAINT IF EXISTS chk_sessions_left_nonnegative;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_user_balance_nonnegative;

DROP TABLE IF EXISTS orders;
DROP INDEX IF EXISTS uq_application_pending_phone;
DROP INDEX IF EXISTS uq_application_pending_email;

-- Restore the v1 cascade behavior for a true rollback.
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_client_id_fkey;
ALTER TABLE payments
    ADD CONSTRAINT payments_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE client_subscriptions DROP CONSTRAINT IF EXISTS client_subscriptions_client_id_fkey;
ALTER TABLE client_subscriptions
    ADD CONSTRAINT client_subscriptions_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE trainings DROP CONSTRAINT IF EXISTS trainings_client_id_fkey;
ALTER TABLE trainings
    ADD CONSTRAINT trainings_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE training_requests ALTER COLUMN status DROP DEFAULT;
ALTER TABLE training_requests
    ALTER COLUMN status TYPE training_status
    USING (
        CASE status::text
            WHEN 'rejected' THEN 'cancelled'
            ELSE status::text
        END
    )::training_status;
ALTER TABLE training_requests ALTER COLUMN status SET DEFAULT 'pending';
DROP TYPE IF EXISTS training_request_status;

ALTER TABLE users DROP COLUMN IF EXISTS token_version;

ALTER TABLE trainings DROP CONSTRAINT IF EXISTS trainings_trainer_id_fkey;
UPDATE trainings tr
SET trainer_id = (
    SELECT t.user_id FROM trainers t WHERE t.id=tr.trainer_id
)
WHERE tr.trainer_id IS NOT NULL;
ALTER TABLE trainings
    ADD CONSTRAINT trainings_trainer_id_fkey
    FOREIGN KEY (trainer_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE trainers DROP CONSTRAINT IF EXISTS uq_trainers_user_id;
ALTER TABLE trainers ALTER COLUMN user_id DROP NOT NULL;

-- The legacy known admin password is intentionally NOT restored on rollback.
