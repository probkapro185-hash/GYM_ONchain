ALTER TABLE training_requests ADD COLUMN trainer_id BIGINT REFERENCES trainers(id) ON DELETE SET NULL;
CREATE INDEX idx_training_requests_trainer ON training_requests(trainer_id);
-- Old requests stay unknown rather than assigning a gender from their name.
ALTER TABLE application_requests ADD COLUMN gender user_gender;
-- Rebuild attendance dates from completed trainings, not the time staff entered them.
UPDATE users u SET last_visit_at=t.last_visit_at
FROM (SELECT client_id,MAX(start_time) AS last_visit_at FROM trainings WHERE status='completed' GROUP BY client_id) t
WHERE u.id=t.client_id;
