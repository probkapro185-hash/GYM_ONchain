-- Normalize legacy training rows to the runtime state machine used by the API.
-- training_status keeps the old enum value for rollback compatibility, but new/changed
-- rows are restricted to scheduled/completed/cancelled by a CHECK constraint below.
UPDATE trainings
SET status='scheduled', updated_at=NOW()
WHERE status='pending';

-- Refuse to guess ownership for malformed legacy rows. These relationships are mandatory
-- in the application model; an operator should repair invalid history explicitly.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM training_requests WHERE client_id IS NULL) THEN
        RAISE EXCEPTION 'cannot enforce training_requests.client_id NOT NULL: legacy NULL rows exist';
    END IF;
    IF EXISTS (SELECT 1 FROM trainings WHERE client_id IS NULL) THEN
        RAISE EXCEPTION 'cannot enforce trainings.client_id NOT NULL: legacy NULL rows exist';
    END IF;
    IF EXISTS (SELECT 1 FROM client_subscriptions WHERE client_id IS NULL OR product_id IS NULL) THEN
        RAISE EXCEPTION 'cannot enforce client_subscriptions ownership: legacy NULL rows exist';
    END IF;
    IF EXISTS (SELECT 1 FROM payments WHERE client_id IS NULL) THEN
        RAISE EXCEPTION 'cannot enforce payments.client_id NOT NULL: legacy NULL rows exist';
    END IF;
END
$$;

ALTER TABLE training_requests ALTER COLUMN client_id SET NOT NULL;
ALTER TABLE trainings ALTER COLUMN client_id SET NOT NULL;
ALTER TABLE client_subscriptions ALTER COLUMN client_id SET NOT NULL;
ALTER TABLE client_subscriptions ALTER COLUMN product_id SET NOT NULL;
ALTER TABLE payments ALTER COLUMN client_id SET NOT NULL;

-- Preserve request history on a direct SQL user deletion too, matching the other
-- training/financial history tables hardened in migration 000003.
ALTER TABLE training_requests DROP CONSTRAINT IF EXISTS training_requests_client_id_fkey;
ALTER TABLE training_requests
    ADD CONSTRAINT training_requests_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE RESTRICT;

-- Enforce that a reserved subscription belongs to the same client as the training.
ALTER TABLE client_subscriptions
    ADD CONSTRAINT uq_client_subscriptions_id_client UNIQUE (id, client_id);
ALTER TABLE trainings DROP CONSTRAINT IF EXISTS trainings_subscription_id_fkey;
ALTER TABLE trainings
    ADD CONSTRAINT trainings_subscription_client_fkey
    FOREIGN KEY (subscription_id, client_id)
    REFERENCES client_subscriptions(id, client_id) ON DELETE RESTRICT;

ALTER TABLE trainings
    ADD CONSTRAINT chk_trainings_runtime_status
    CHECK (status IN ('scheduled','completed','cancelled'));

-- A fresh/changed scheduled training must reserve a subscription. NOT VALID keeps a
-- migration from failing solely because an old scheduled row cannot be mapped safely;
-- PostgreSQL still enforces the constraint for all future inserts/updates.
ALTER TABLE trainings
    ADD CONSTRAINT chk_scheduled_training_has_subscription
    CHECK (status <> 'scheduled' OR subscription_id IS NOT NULL) NOT VALID;
