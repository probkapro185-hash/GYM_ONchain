# Реализация AI/RAG

## Архитектура

```text
Browser / CRM frontend
        |
        v
POST /api/v1/ai/chat
        |
        +--> JWT + актуальный user/role из PostgreSQL
        |
        +--> embedding вопроса
        |       |
        |       v
        |   pgvector cosine search
        |       |
        |       v
        |   релевантные knowledge chunks
        |
        +--> безопасный персональный контекст текущего пользователя
        |
        +--> OpenAI Responses API
        |
        +--> транзакционно сохраняются user message + assistant message + RAG sources
```

## Что хранится в PostgreSQL

Миграции `000004` и `000005` создают/расширяют:

- `ai_documents` — документ, source, checksum;
- `ai_chunks` — чанки и `vector(1536)` embeddings;
- `ai_conversations` — диалоги пользователя;
- `ai_messages` — сообщения и JSONB `sources`.

HNSW-индекс используется для cosine nearest-neighbor поиска.

## Индексация

`IndexKnowledge` читает `.md/.txt` из `AI_KNOWLEDGE_DIR`.

- SHA-256 не даёт переиндексировать неизменённый файл;
- изменённый документ заменяет свои чанки транзакционно;
- удалённый/пустой файл удаляется и из БД;
- reindex сериализован `indexMu`, чтобы два admin-запроса не переписывали индекс одновременно;
- embeddings проверяются по количеству, индексу, размерности и на NaN/Inf;
- chunker не создаёт overlap-only дубликаты на длинных абзацах.

## Контекст ответа

RAG используется для стабильных знаний о CRM: навигация, роли, расписание, магазин, правила работы.

Актуальные пользовательские значения не индексируются. Для роли `client` backend отдельно получает из основной БД:

- имя и роль;
- баланс и visits;
- активный абонемент и его историческое название;
- sessions left / end date;
- до пяти ближайших scheduled тренировок.

Роль берётся из записи пользователя в БД. Переданный клиентом role/user_id не используется как источник доверия.

## История и sources

Диалог принадлежит конкретному `user_id`. Чужой conversation ID не открывается. Для model context используются последние 12 сообщений, а endpoint просмотра истории возвращает последние 100.

RAG sources каждого assistant message сохраняются в `ai_messages.sources`; UI показывает их и сразу после ответа, и после повторного открытия истории.

## OpenAI client hardening

- backend использует `/v1/embeddings` и `/v1/responses`;
- API key никогда не передаётся во frontend;
- response body ограничен при чтении;
- HTTP client имеет timeout;
- embeddings валидируются до записи в pgvector;
- Responses parser обрабатывает и `output_text`, и явный `refusal`;
- `store=false`;
- `safety_identifier` — `sfedu_` + SHA-256 от внутреннего user ID namespace, без email/телефона.

## Prompt-injection boundary

System instructions явно говорят, что RAG chunks являются данными, а не командами. Модель не должна исполнять инструкции внутри knowledge-документов, раскрывать секреты или придумывать функции CRM.

Это дополнительная защита, а не замена authorization: доступ к персональным данным отсекается в backend до обращения к модели.
