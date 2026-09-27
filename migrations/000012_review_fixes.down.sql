ALTER TABLE application_requests DROP COLUMN gender;
DROP INDEX idx_training_requests_trainer;
ALTER TABLE training_requests DROP COLUMN trainer_id;
-- Corrected attendance dates are retained on rollback.
