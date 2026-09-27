CREATE TYPE user_role AS ENUM ('client', 'manager', 'admin');
CREATE TYPE user_gender AS ENUM ('male', 'female');
CREATE TYPE training_status AS ENUM ('pending', 'scheduled', 'completed', 'cancelled');
CREATE TYPE operation_type AS ENUM ('income', 'expense', 'refund');
CREATE TYPE service_type AS ENUM ('subscription', 'training', 'product', 'deposit');
CREATE TYPE product_category AS ENUM ('subscription', 'sports');
CREATE TYPE subscription_type AS ENUM ('monthly', 'quarterly', 'annual', 'single');
CREATE TYPE trainer_specialization AS ENUM ('body_relief', 'weight_loss', 'mass_gain');
CREATE TYPE application_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    full_name     VARCHAR(255) NOT NULL,
    phone         VARCHAR(20)  NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          user_role    NOT NULL DEFAULT 'client',
    gender        user_gender  NOT NULL DEFAULT 'male',
    balance       NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    visits        INTEGER NOT NULL DEFAULT 0,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_visit_at TIMESTAMPTZ
);
CREATE INDEX idx_users_role ON users(role);

CREATE TABLE application_requests (
    id         BIGSERIAL PRIMARY KEY,
    full_name  VARCHAR(255) NOT NULL,
    phone      VARCHAR(20) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    status     application_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_app_requests_status ON application_requests(status);

CREATE TABLE trainers (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT REFERENCES users(id) ON DELETE CASCADE,
    specialization   trainer_specialization NOT NULL,
    bio              TEXT,
    photo_url        VARCHAR(500),
    experience_years INTEGER NOT NULL DEFAULT 0,
    is_active        BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_trainers_specialization ON trainers(specialization);
CREATE INDEX idx_trainers_user_id ON trainers(user_id);

CREATE TABLE training_requests (
    id           BIGSERIAL PRIMARY KEY,
    client_id    BIGINT REFERENCES users(id) ON DELETE CASCADE,
    preferred_at TIMESTAMPTZ NOT NULL,
    comment      TEXT,
    status       training_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_training_requests_client ON training_requests(client_id);
CREATE INDEX idx_training_requests_status ON training_requests(status);

-- Kept compatible with the original v1 schema. Migration 000003 converts
-- trainer_id from users(id) to trainers(id) for both fresh and existing DBs.
CREATE TABLE trainings (
    id          BIGSERIAL PRIMARY KEY,
    client_id   BIGINT REFERENCES users(id) ON DELETE CASCADE,
    trainer_id  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    start_time  TIMESTAMPTZ NOT NULL,
    end_time    TIMESTAMPTZ NOT NULL,
    status      training_status NOT NULL DEFAULT 'scheduled',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_training_time CHECK (end_time > start_time)
);
CREATE INDEX idx_trainings_client ON trainings(client_id);
CREATE INDEX idx_trainings_trainer ON trainings(trainer_id);
CREATE INDEX idx_trainings_start ON trainings(start_time);
CREATE INDEX idx_trainings_status ON trainings(status);

CREATE TABLE products (
    id             BIGSERIAL PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    price          NUMERIC(12,2) NOT NULL,
    category       product_category NOT NULL,
    sub_type       subscription_type,
    duration_days  INTEGER,
    sessions_count INTEGER,
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_price_positive CHECK (price > 0)
);
CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_active ON products(is_active);

CREATE TABLE client_subscriptions (
    id            BIGSERIAL PRIMARY KEY,
    client_id     BIGINT REFERENCES users(id) ON DELETE CASCADE,
    product_id    BIGINT REFERENCES products(id) ON DELETE RESTRICT,
    start_date    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    end_date      TIMESTAMPTZ NOT NULL,
    sessions_left INTEGER,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sub_dates CHECK (end_date > start_date)
);
CREATE INDEX idx_subscriptions_client ON client_subscriptions(client_id);
CREATE INDEX idx_subscriptions_active ON client_subscriptions(is_active);

CREATE TABLE payments (
    id             BIGSERIAL PRIMARY KEY,
    client_id      BIGINT REFERENCES users(id) ON DELETE CASCADE,
    amount         NUMERIC(12,2) NOT NULL,
    operation_type operation_type NOT NULL,
    service_type   service_type NOT NULL,
    description    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_amount_positive CHECK (amount > 0)
);
CREATE INDEX idx_payments_client ON payments(client_id);
CREATE INDEX idx_payments_operation ON payments(operation_type);
CREATE INDEX idx_payments_created ON payments(created_at);
