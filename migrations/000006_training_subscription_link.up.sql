-- Bind each training to the exact subscription that reserves/pays for the session.
-- The column stays nullable only for legacy rows that cannot be mapped safely.
ALTER TABLE trainings ADD COLUMN subscription_id BIGINT;

UPDATE trainings tr
SET subscription_id = (
    SELECT cs.id
    FROM client_subscriptions cs
    WHERE cs.client_id = tr.client_id
      AND cs.start_date <= tr.start_time
      AND cs.end_date >= tr.end_time
    ORDER BY cs.end_date ASC, cs.id ASC
    LIMIT 1
)
WHERE tr.subscription_id IS NULL;

ALTER TABLE trainings
    ADD CONSTRAINT trainings_subscription_id_fkey
    FOREIGN KEY (subscription_id) REFERENCES client_subscriptions(id) ON DELETE RESTRICT;

CREATE INDEX idx_trainings_subscription ON trainings(subscription_id);
