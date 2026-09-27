-- Never seed a known/default password. The application replaces this marker
-- with a bcrypt hash of ADMIN_PASSWORD before the HTTP server starts.
INSERT INTO users (full_name, phone, email, password_hash, role, gender, is_active)
VALUES (
    'Администратор SFEDU',
    '+70000000000',
    'admin@gmail.com',
    'BOOTSTRAP_REQUIRED',
    'admin',
    'male',
    false
)
ON CONFLICT (email) DO NOTHING;
