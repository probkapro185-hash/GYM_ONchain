ALTER TABLE client_subscriptions
    ADD COLUMN frozen_at TIMESTAMPTZ,
    ADD COLUMN freeze_days INTEGER NOT NULL DEFAULT 0 CHECK (freeze_days >= 0);

CREATE TABLE client_progress (
    id BIGSERIAL PRIMARY KEY,
    client_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    recorded_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    weight NUMERIC(6,2),
    body_fat NUMERIC(5,2),
    waist NUMERIC(6,2),
    chest NUMERIC(6,2),
    hips NUMERIC(6,2),
    comment TEXT,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_progress_weight CHECK (weight IS NULL OR weight > 0),
    CONSTRAINT chk_progress_body_fat CHECK (body_fat IS NULL OR body_fat BETWEEN 0 AND 100),
    CONSTRAINT chk_progress_waist CHECK (waist IS NULL OR waist > 0),
    CONSTRAINT chk_progress_chest CHECK (chest IS NULL OR chest > 0),
    CONSTRAINT chk_progress_hips CHECK (hips IS NULL OR hips > 0)
);
CREATE INDEX idx_client_progress_client_time ON client_progress(client_id, recorded_at DESC);

CREATE TABLE client_notes (
    id BIGSERIAL PRIMARY KEY,
    client_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    note TEXT NOT NULL CHECK (char_length(note) BETWEEN 1 AND 4000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_client_notes_client_time ON client_notes(client_id, created_at DESC);

CREATE TABLE staff_tasks (
    id BIGSERIAL PRIMARY KEY,
    client_id BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    assignee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    due_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open','done','cancelled')),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_staff_tasks_assignee_status_due ON staff_tasks(assignee_id, status, due_at);
CREATE INDEX idx_staff_tasks_client ON staff_tasks(client_id);

CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_user_read_time ON notifications(user_id, is_read, created_at DESC);

CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    actor_role VARCHAR(32),
    request_id VARCHAR(80) NOT NULL,
    method VARCHAR(12) NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_log_time ON audit_log(created_at DESC);
CREATE INDEX idx_audit_log_actor ON audit_log(actor_id, created_at DESC);
