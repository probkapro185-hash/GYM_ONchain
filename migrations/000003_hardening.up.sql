-- 1) Neutralize the legacy public bootstrap password from the original repository.
UPDATE users
SET password_hash='BOOTSTRAP_REQUIRED', is_active=false, updated_at=NOW()
WHERE lower(email)='admin@gmail.com'
  AND password_hash='$2a$12$obOWT7.qHQyoG2qLCUuvW.IybKvUfxobSFa3Ps2VdxtJIBJPmDVfy';


-- Existing tokens from older releases do not contain token_version (they decode as 0).
-- Start at 1 so all pre-hardening JWTs are invalidated on deployment.
ALTER TABLE users
    ADD COLUMN token_version INTEGER NOT NULL DEFAULT 1 CHECK (token_version > 0);

-- Training-request workflow is not the same state machine as an actual training.
CREATE TYPE training_request_status AS ENUM ('pending', 'scheduled', 'rejected');
ALTER TABLE training_requests ALTER COLUMN status DROP DEFAULT;
ALTER TABLE training_requests
    ALTER COLUMN status TYPE training_request_status
    USING (
        CASE status::text
            WHEN 'cancelled' THEN 'rejected'
            WHEN 'completed' THEN 'scheduled'
            ELSE status::text
        END
    )::training_request_status;
ALTER TABLE training_requests ALTER COLUMN status SET DEFAULT 'pending';

-- 2) A user can have at most one trainer profile. If historical duplicates exist,
-- keep the oldest active-ish profile; trainings still point to users at this point.
DELETE FROM trainers t
USING (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY user_id
               ORDER BY is_active DESC, created_at ASC, id ASC
           ) AS rn
    FROM trainers
    WHERE user_id IS NOT NULL
) d
WHERE t.id=d.id AND d.rn>1;

DELETE FROM trainers WHERE user_id IS NULL;

ALTER TABLE trainers ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE trainers
    ADD CONSTRAINT uq_trainers_user_id UNIQUE (user_id);

-- 3) Fix the original structural FK bug:
-- trainings.trainer_id used to contain users.id; the domain/API use trainers.id.
ALTER TABLE trainings DROP CONSTRAINT IF EXISTS trainings_trainer_id_fkey;

UPDATE trainings tr
SET trainer_id = (
    SELECT t.id
    FROM trainers t
    WHERE t.user_id = tr.trainer_id
    ORDER BY t.id
    LIMIT 1
)
WHERE tr.trainer_id IS NOT NULL;

ALTER TABLE trainings
    ADD CONSTRAINT trainings_trainer_id_fkey
    FOREIGN KEY (trainer_id) REFERENCES trainers(id) ON DELETE SET NULL;

-- 4) Avoid duplicate public applications that race each other.
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY lower(email) ORDER BY created_at DESC, id DESC) AS rn
    FROM application_requests
    WHERE status='pending'
)
UPDATE application_requests a
SET status='rejected'
FROM ranked r
WHERE a.id=r.id AND r.rn>1;

WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY phone ORDER BY created_at DESC, id DESC) AS rn
    FROM application_requests
    WHERE status='pending'
)
UPDATE application_requests a
SET status='rejected'
FROM ranked r
WHERE a.id=r.id AND r.rn>1;

CREATE UNIQUE INDEX uq_application_pending_email
    ON application_requests (lower(email))
    WHERE status='pending';
CREATE UNIQUE INDEX uq_application_pending_phone
    ON application_requests (phone)
    WHERE status='pending';

-- 5) Preserve financial/training history even if someone issues a direct SQL DELETE.
ALTER TABLE trainings DROP CONSTRAINT IF EXISTS trainings_client_id_fkey;
ALTER TABLE trainings
    ADD CONSTRAINT trainings_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE client_subscriptions DROP CONSTRAINT IF EXISTS client_subscriptions_client_id_fkey;
ALTER TABLE client_subscriptions
    ADD CONSTRAINT client_subscriptions_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_client_id_fkey;
ALTER TABLE payments
    ADD CONSTRAINT payments_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE RESTRICT;

-- 6) Sports-product purchases now create a durable order record.
CREATE TABLE orders (
    id         BIGSERIAL PRIMARY KEY,
    client_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    amount     NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    status     VARCHAR(20) NOT NULL DEFAULT 'paid'
               CHECK (status IN ('pending','paid','cancelled','refunded')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_orders_client ON orders(client_id);
CREATE INDEX idx_orders_created ON orders(created_at);

-- 7) Enforce invariants for all new/changed rows while allowing a legacy DB
-- containing older bad data to migrate successfully (NOT VALID skips the old scan).
ALTER TABLE users
    ADD CONSTRAINT chk_user_balance_nonnegative CHECK (balance >= 0) NOT VALID;

ALTER TABLE client_subscriptions
    ADD CONSTRAINT chk_sessions_left_nonnegative
    CHECK (sessions_left IS NULL OR sessions_left >= 0) NOT VALID;

ALTER TABLE products
    ADD CONSTRAINT chk_product_shape
    CHECK (
        (category='sports' AND sub_type IS NULL AND duration_days IS NULL AND sessions_count IS NULL)
        OR
        (category='subscription' AND sub_type IS NOT NULL AND duration_days > 0
         AND (sessions_count IS NULL OR sessions_count > 0))
    ) NOT VALID;
