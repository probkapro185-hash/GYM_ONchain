ALTER TABLE ai_messages
    ADD COLUMN sources JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE ai_messages
    ADD CONSTRAINT chk_ai_message_sources_array
    CHECK (jsonb_typeof(sources) = 'array');
