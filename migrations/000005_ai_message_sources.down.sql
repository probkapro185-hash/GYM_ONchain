ALTER TABLE ai_messages DROP CONSTRAINT IF EXISTS chk_ai_message_sources_array;
ALTER TABLE ai_messages DROP COLUMN IF EXISTS sources;
