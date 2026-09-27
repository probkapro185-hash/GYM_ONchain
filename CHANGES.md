# Полный список основных исправлений

## Исправления по проверке проекта (2026-09-24)

Исправлены сборка, начальная настройка администратора, параметры почты и хранение фото в Docker, сохранение тренера в заявке, календарь, состояния и резервирование абонементов, права на задачи, обход активации, CSV, уведомления и статистика посещений. Дополнительно исправлены выбор пола и типа абонемента, видимость экспорта и восстановление тренеров. Добавлена миграция 000012 и регрессионные тесты.

Подробности: [REVIEW_FIXES.md](REVIEW_FIXES.md). Результаты проверки: [VERIFICATION.md](VERIFICATION.md).

## Account activation + password recovery (2026-09-20)

- staff no longer chooses a password when approving a public client application;
- approved/invited clients are created in pending activation state;
- added cryptographically random one-time activation and password-reset links;
- only SHA-256 token hashes are stored in PostgreSQL;
- activation links are single-use and expire after configurable TTL;
- added «Забыли пароль?» flow with account-enumeration-safe response;
- password reset invalidates previous JWTs via `token_version`;
- manager/admin can resend an activation invitation;
- staff-created clients use the same self-service activation flow;
- added SMTP delivery and safe development `MAIL_MODE=log`; production requires SMTP;
- added migration `000011_account_access`;
- added service/mailer/config tests and frontend regression guards;
- AI/RAG implementation files were not modified.

## CRM+ — расширение без изменения AI/RAG (2026-09-15)

- frontend разделён на HTML/CSS/JS assets и продолжает обслуживаться тем же Go backend;
- добавлен дневник физического прогресса клиента;
- добавлены staff-only заметки по клиенту;
- добавлены follow-up задачи с дедлайнами, просрочкой и отдельным экраном «Мои задачи»;
- добавлены retention-сигналы: 14+ дней без визита, скорое окончание абонемента, открытые задачи;
- добавлена заморозка/разморозка абонемента с переносом даты окончания на срок заморозки;
- добавлено ручное продление абонемента;
- заморозка запрещена при будущих запланированных тренировках;
- добавлена клиентская отмена тренировки с окном 2 часа без списания занятия;
- добавлены in-app уведомления;
- добавлена «продажа на стойке» через manager/admin с транзакционным финансовым следом;
- добавлен admin dashboard: клиенты, абонементы, заявки, удержание, поступления, тренировки и загрузка тренеров;
- добавлены CSV-экспорты клиентов, финансов и CRM-карточки;
- добавлен audit log изменяющих запросов;
- добавлен безопасный `X-Request-ID` и корреляция с логами/аудитом;
- добавлены `/readyz` и Prometheus-compatible `/metrics`;
- добавлены security headers/CSP;
- добавлены unit-тесты для CRM validation и observability middleware;
- добавлен `scripts/verify_frontend.py` для статической проверки frontend integration;
- добавлена миграция `000010_crm_expansion`;
- AI/RAG service, OpenAI client и AI handler намеренно не изменялись.

## Security / auth

- удалён рабочий `.env`, добавлены `.env.example`, `.gitignore`, `.dockerignore`;
- старые опубликованные секреты больше не входят в проект;
- `.env` необязателен для Docker/production;
- удалён известный admin password, безопасный bootstrap через `ADMIN_PASSWORD`;
- HS256 + issuer + expiration + `token_version`;
- пользователь/роль перечитываются из БД на каждом защищённом запросе;
- смена password/activity инвалидирует старые JWT;
- закрыта `manager -> admin` privilege escalation;
- закрыт IDOR просмотра чужой тренировки;
- CORS переведён на allowlist;
- внутренние DB errors не отдаются клиенту;
- JSON body limit 1 MiB, unknown fields запрещены;
- rate limits: login, public application, AI chat.

## Users / data normalization

- manager ограничен client accounts;
- delete user заменён на soft deactivate для сохранения истории;
- пароли валидируются во всех flows;
- телефоны нормализуются в `+7XXXXXXXXXX`;
- миграция отклоняет неоднозначные/некорректные legacy phone identities вместо опасного merge;
- email приводится к lower/trim и защищён case-insensitive uniqueness.

## Finance / shop

- удалён self-topup клиента;
- staff topup/purchase/payment/order/subscription выполняются транзакционно;
- суммы перед DB writes переводятся в integer cents;
- RowsAffected проверяется для update/delete flows;
- нельзя купить inactive product;
- sports purchase создаёт durable `orders` record;
- subscriptions/orders сохраняют historical product name; subscription хранит purchase price snapshot;
- позднее редактирование product не изменяет историю покупки.

## Schedule / subscriptions

- request и training получили разные status state machines;
- approve request всегда использует owner заявки;
- активный абонемент проверяется по времени и sessions left;
- `sessions_left=0` не считается активным;
- каждое занятие резервирует конкретный `subscription_id`;
- занятие не может использовать subscription другого client (включая DB composite FK);
- scheduled sessions учитываются как reserved до completed/cancelled, даже если время уже прошло;
- completion списывает именно зарезервированный subscription и увеличивает visits в одной транзакции;
- проверяются пересечения клиента/тренера;
- тренировка не планируется за пределами срока subscription;
- legacy actual training status `pending` мигрируется в `scheduled`;
- усилены NOT NULL/FK/DB checks для ownership.

## Trainers / schema

- исправлен исходный FK `trainings.trainer_id`: users -> trainers;
- `trainers.user_id` стал unique/not null;
- trainer delete стал deactivate;
- user changes инвалидируют и users cache, и trainers cache.

## AI / RAG

- добавлены pgvector tables, HNSW cosine index, conversations/messages;
- RAG sources сохраняются в истории (`ai_messages.sources` JSONB);
- удалённые/пустые knowledge files удаляются из vector index;
- reindex защищён от параллельного запуска внутри процесса;
- embeddings валидируются: count/index/dimension/NaN/Inf;
- исправлен chunker для длинных абзацев без overlap-only duplicates;
- динамические персональные данные берутся из основной БД, а не embeddings;
- DB role является source of truth;
- OpenAI `safety_identifier` строится приватно из internal user ID;
- Responses parser поддерживает `output_text` и `refusal`;
- `store=false`, API key остаётся только backend;
- server write timeout увеличен для AI calls.

## Docker / runtime

- исправлен fatal `.env` startup bug;
- Docker final image запускается non-root;
- migrations/knowledge копируются в runtime image;
- Redis действительно необязателен для app startup;
- PostgreSQL image содержит pgvector;
- обновлены прямые Go-зависимости: pgx 5.10.0, jwt 5.3.1, migrate 4.19.1, go-redis 9.22.0, x/crypto 0.51.0;
- минимальный Go поднят до 1.25 из-за актуальных зависимостей;
- Docker builder/runtime обновлены;
- DB startup context и admin bootstrap context разделены, чтобы длительная migration не истекала перед bootstrap.

## Frontend HTML

- полноценный старый SFEDU frontend перенесён в `frontend/index.html` и подключён к текущему Go backend;
- backend теперь раздаёт основной сайт на `/`;
- frontend использует same-origin `/api/v1` вместо жёсткого localhost URL;
- login автоматически сохраняет JWT и использует его для всех защищённых запросов;
- встроенный «ИИ Спортсмен» переведён на RAG contract `{message, conversation_id?}` и больше не отправляет `user_id`;
- удалены зависимости UI от небезопасного `/finance/me/topup`;

- исправлена mobile navigation;
- устранены duplicate send/race conditions;
- динамический HTML использует централизованное экранирование пользовательских значений; `eval` отсутствует;
- URL/storage/error guards;
- улучшены accessibility/focus/contrast;
- sources отображаются и после повторного открытия истории.

## Миграции

Финальная последовательность: `000001` ... `000011`. Подробное назначение каждой версии описано в `README.md`.

## 2026-09-15 — expanded tests + UI polish

### Test coverage

- test suite expanded to 23 Go test files and 100 `Test*`/`Fuzz*` functions (including optional integration smoke tests);
- added broad service tests for auth, users/RBAC, trainers, schedule/subscription reservations, finance/shop and CRM+;
- added middleware tests for authentication, stale JWT, role guards, rate limiting, security headers and metrics;
- added strict HTTP decoding/filter/error-mapping tests;
- added PostgreSQL helper/query-construction tests;
- added fuzz targets for phone normalization, money conversion and full-name validation;
- added frontend API/DOM/security regression checker;
- added migration-pair and lightweight committed-secret checks;
- added optional PostgreSQL integration smoke tests and HTTP E2E smoke script;
- Makefile now exposes shuffled tests, race detector, coverage gate, fuzz, frontend, static, integration and E2E targets.

### Frontend polish

The visual identity remains SFEDU dark/navy + electric blue. No redesign into another style was performed. Improvements include softer navy surfaces, restrained glass/backdrop effects, stronger visual hierarchy, cleaner cards/tables/calendars, gradient primary actions, clearer keyboard focus, responsive refinements and reduced-motion accessibility support.

AI/RAG implementation files were hash-checked against the previous CRM+ build and remain unchanged.

## Frontend polish: staff navigation and protected admin profile

- Replaced emoji icons for **Dashboard** and **Мои задачи** in the staff sidebar with consistent monochrome SVG icons.
- Admin profile now opens in read-only mode: name, phone, and email cannot be edited until **Изменить данные** is pressed.
- Added explicit **Сохранить** and **Отмена** edit controls. Cancelling restores the currently saved account data.
- Reopening the admin profile returns it to protected read-only mode.
- Managers keep the existing editable profile behavior.
- Added frontend verification assertions for the new profile controls and non-emoji Dashboard/Tasks navigation.
- AI/RAG implementation was not changed.

## Trainer assignment + compact local photos

- Trainer creation now explains the assignment model: an admin assigns a trainer profile to an existing active user; users already assigned as active trainers are hidden from the picker.
- Trainer photos can be selected directly from local files instead of requiring an external URL.
- The browser center-crops and resizes the image to at most 320×320, prefers WebP, and progressively lowers quality/size before upload. Typical output is roughly 15–50 KB; the frontend rejects output above 96 KB.
- The backend independently enforces a 96 KiB final-image limit and accepts only compressed WebP/JPEG payloads.
- Uploaded photos are stored as small runtime files under `data/uploads/trainers/`; PostgreSQL stores only the short relative path, never base64 image data.
- Random immutable filenames and one-year browser caching reduce repeated traffic. Replacing/removing a managed photo deletes the previous local file so storage does not grow unnecessarily.
- Trainer photo serving rejects arbitrary filenames/path traversal and does not expose directory listing.
- No database migration is required; the existing `photo_url` column stores the relative managed path.
