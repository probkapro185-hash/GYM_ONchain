-- Canonicalize Russian phone numbers so equivalent formats cannot bypass uniqueness.
-- Fail loudly if legacy data contains an invalid or duplicate user phone; silently
-- merging user identities would be unsafe.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM users
        WHERE btrim(phone) !~ '^(\+7|7|8)[0-9 ()-]*$'
           OR CASE
                WHEN regexp_replace(phone, '[^0-9]', '', 'g') ~ '^8[0-9]{10}$'
                    THEN '+7' || substr(regexp_replace(phone, '[^0-9]', '', 'g'), 2)
                WHEN regexp_replace(phone, '[^0-9]', '', 'g') ~ '^7[0-9]{10}$'
                    THEN '+' || regexp_replace(phone, '[^0-9]', '', 'g')
                ELSE NULL
              END IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot canonicalize legacy users.phone: invalid Russian phone exists';
    END IF;

    IF EXISTS (
        SELECT canonical_phone
        FROM (
            SELECT CASE
                WHEN regexp_replace(phone, '[^0-9]', '', 'g') LIKE '8%'
                    THEN '+7' || substr(regexp_replace(phone, '[^0-9]', '', 'g'), 2)
                ELSE '+' || regexp_replace(phone, '[^0-9]', '', 'g')
            END AS canonical_phone
            FROM users
        ) normalized
        GROUP BY canonical_phone
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot canonicalize users.phone: duplicate users resolve to the same phone';
    END IF;
END $$;

UPDATE users
SET phone = CASE
    WHEN regexp_replace(phone, '[^0-9]', '', 'g') LIKE '8%'
        THEN '+7' || substr(regexp_replace(phone, '[^0-9]', '', 'g'), 2)
    ELSE '+' || regexp_replace(phone, '[^0-9]', '', 'g')
END;

-- Email identity is case-insensitive in the application. Canonicalize legacy rows
-- and enforce that rule in the database as well.
DO $$
BEGIN
    IF EXISTS (
        SELECT lower(btrim(email))
        FROM users
        GROUP BY lower(btrim(email))
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot canonicalize users.email: case-insensitive duplicate users exist';
    END IF;
END $$;

UPDATE users SET email=lower(btrim(email));
CREATE UNIQUE INDEX uq_users_email_lower ON users (lower(email));
ALTER TABLE users
    ADD CONSTRAINT chk_users_email_canonical CHECK (email=lower(email) AND email=btrim(email));

DROP INDEX IF EXISTS uq_application_pending_email;
UPDATE application_requests SET email=lower(btrim(email));
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY email ORDER BY created_at DESC, id DESC) AS rn
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
ALTER TABLE application_requests
    ADD CONSTRAINT chk_application_email_canonical CHECK (email=lower(email) AND email=btrim(email));

-- The old partial index compares raw strings, so rebuild it after normalization.
DROP INDEX IF EXISTS uq_application_pending_phone;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM application_requests
        WHERE btrim(phone) !~ '^(\+7|7|8)[0-9 ()-]*$'
           OR CASE
                WHEN regexp_replace(phone, '[^0-9]', '', 'g') ~ '^8[0-9]{10}$'
                    THEN '+7' || substr(regexp_replace(phone, '[^0-9]', '', 'g'), 2)
                WHEN regexp_replace(phone, '[^0-9]', '', 'g') ~ '^7[0-9]{10}$'
                    THEN '+' || regexp_replace(phone, '[^0-9]', '', 'g')
                ELSE NULL
              END IS NULL
    ) THEN
        RAISE EXCEPTION 'cannot canonicalize legacy application_requests.phone: invalid Russian phone exists';
    END IF;
END $$;

UPDATE application_requests
SET phone = CASE
    WHEN regexp_replace(phone, '[^0-9]', '', 'g') LIKE '8%'
        THEN '+7' || substr(regexp_replace(phone, '[^0-9]', '', 'g'), 2)
    ELSE '+' || regexp_replace(phone, '[^0-9]', '', 'g')
END;

-- If two pending applications used two visual formats of one phone, keep the newest.
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

CREATE UNIQUE INDEX uq_application_pending_phone
    ON application_requests (phone)
    WHERE status='pending';

ALTER TABLE users
    ADD CONSTRAINT chk_users_phone_canonical
    CHECK (phone ~ '^\+7[0-9]{10}$');

ALTER TABLE application_requests
    ADD CONSTRAINT chk_application_phone_canonical
    CHECK (phone ~ '^\+7[0-9]{10}$');
