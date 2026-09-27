ALTER TABLE application_requests DROP CONSTRAINT IF EXISTS chk_application_email_canonical;
ALTER TABLE application_requests DROP CONSTRAINT IF EXISTS chk_application_phone_canonical;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_email_canonical;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_phone_canonical;
DROP INDEX IF EXISTS uq_users_email_lower;
-- Phone/email canonicalization itself is intentionally not reversed: original
-- formatting/case is presentation-only information and cannot be reconstructed losslessly.
