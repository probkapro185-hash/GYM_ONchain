DROP TABLE IF EXISTS account_tokens;
DROP TYPE IF EXISTS account_token_purpose;
ALTER TABLE users DROP COLUMN IF EXISTS password_setup_required;
