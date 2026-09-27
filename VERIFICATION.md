# Verification report

## Проверка исправлений 2026-09-24

В рабочей копии выполнены полные проверки, которые ранее были недоступны:

- `make test-all` — успешно: форматирование, миграции (12 пар), проверка секретов, HTML/API-контракты, синтаксис JS, 8 поведенческих тестов frontend, `go vet`, все Go-тесты в случайном порядке.
- `make test-race` — успешно. После последующих изменений повторно проверены затронутые service/postgres пакеты с `-race`.
- `go test -tags=integration -count=1 ./integration/...` — успешно на отдельной PostgreSQL 16 с pgvector. Проверены реальные SQL-запросы бронирования, переноса, тренеров, задач, посещений, пола заявки, заморозки и AI-контекста без вызовов внешнего AI API.
- Миграции `1 → 12` с пустой БД и `11 → 12` с историческими данными — успешно; отдельно проверены пересчёт даты визита и сохранение старой заявки без выдуманного пола.
- Docker-образ собирается; Compose передаёт все параметры почты и подключает volume фотографий.
- Приложение запущено в Docker вместе с отдельными PostgreSQL и Redis. Через HTTP проверены вход администратора с собственным email, запрет обхода активации, права на задачи, заявка → одобрение → активация → вход клиента, свежий баланс при включённом Redis, числовой CSV, восстановление тренера и реальная загрузка/чтение JPEG.
- После пересоздания контейнера загруженное фото сохранилось, пароль уже созданного администратора не изменился при смене bootstrap-настройки.

Все проверки выполнялись на временных тестовых данных. Рабочая база пользователя не изменялась. Письма использовали `MAIL_MODE=log`; доставка через реальный SMTP-провайдер и внешние AI-запросы не выполнялись.

Список изменений и запуск обновлённого проекта: [REVIEW_FIXES.md](REVIEW_FIXES.md).

## Исторический отчёт исходной версии

Дата проверки TESTED-UI версии: **2026-09-15**.

## Добавленные уровни тестирования

- **23 Go test files**;
- **100 Test/Fuzz functions**, включая 3 optional PostgreSQL integration smoke tests;
- service/business-rule tests: auth, users/RBAC, trainers, schedule, subscriptions, finance/shop, CRM+;
- middleware/auth/rate-limit/security-header/metrics tests;
- strict HTTP decode, path/date/filter/error mapping tests;
- PostgreSQL error/query helper tests;
- fuzz tests for phone, money and full name input;
- frontend DOM/API contract tests;
- migration and basic secret-regression checks;
- optional E2E smoke script;
- race/coverage/shuffle/fuzz Makefile targets.

## Проверки, выполненные в sandbox

- `gofmt` успешно разобрал и отформатировал **66 Go files**; синтаксических ошибок при форматировании нет;
- `python3 scripts/verify_frontend.py` — **PASS**: 247 unique HTML ids, CRM+ wiring present, AI contract unchanged;
- `python3 scripts/test_frontend_contract.py` — **PASS**: frontend contract, 11 critical API routes;
- `node --check frontend/assets/app.js` — **PASS**;
- `python3 scripts/check_migrations.py` — **PASS**: 10 ordered up/down migration pairs (`000001`–`000010`);
- `python3 scripts/check_secrets.py` — **PASS**: obvious committed-secret patterns not detected;
- Python scripts compile and `scripts/e2e_smoke.sh` passes shell syntax validation;
- validator unit tests were executed in an isolated local Go 1.23-compatible harness and **PASS**;
- all three validator fuzz targets were executed for a short smoke run and **PASS**, with tens of thousands of generated inputs;
- AI-critical implementation files were SHA-256 compared with the previous CRM+ version and are **unchanged**.

## Sandbox limitation for the full Go suite

The project declares Go **1.25.0**. The sandbox cannot download that toolchain because outbound DNS/network access to `proxy.golang.org` is blocked. A direct `go test ./...` attempt stops before project compilation while downloading Go 1.25. This is an environment limitation, not a passing result; therefore this report does **not** claim that the complete Go suite was executed here.

Run the complete suite on the developer Mac/CI with Go 1.25+:

```bash
make test-static
make test-frontend
make test-shuffle
make test-race
make coverage-check
```

Optional database integration:

```bash
TEST_DATABASE_URL='postgres://...' make integration
```

Optional running-server smoke:

```bash
make e2e
# authenticated:
E2E_EMAIL='...' E2E_PASSWORD='...' make e2e
```

## AI/RAG preservation

These implementation files were deliberately not modified in this revision:

- `internal/service/ai_assistant.go`
- `internal/handler/http/ai.go`
- `pkg/openai/client.go`
- `examples/ai-chat/index.html`
- `knowledge/06_ai_assistant.md`

The main frontend retains the same existing `{message, conversation_id?}` RAG API contract; only its surrounding visual CSS was polished as part of the overall UI update.

## Account access update — 2026-09-20

- `python3 scripts/check_migrations.py` — PASS: 11 ordered migration pairs (`000001`–`000011`).
- `python3 scripts/check_secrets.py` — PASS.
- `python3 scripts/verify_frontend.py` — PASS: activation/reset/invite frontend wiring present; AI contract unchanged.
- `python3 scripts/test_frontend_contract.py` — PASS: client password is no longer collected during application approval.
- `node --check frontend/assets/app.js` — PASS.
- `go test ./pkg/mailer` — PASS using the locally available Go toolchain after a temporary go-directive compatibility check; final `go.mod` remains Go 1.25.
- `go test ./internal/config` — PASS under the same isolated compatibility check.
- Full `go test ./...` could not be executed in this sandbox because external Go modules are not cached and outbound access to `proxy.golang.org` is blocked. Run it on the developer Mac where dependencies are available.

## Frontend profile/navigation update

- `python3 scripts/verify_frontend.py` — PASS (262 unique HTML ids; admin profile edit-mode checks included)
- `node --check frontend/assets/app.js` — PASS
- `python3 scripts/test_frontend_contract.py` — PASS
- `python3 scripts/check_migrations.py` — PASS (migrations 000001–000011)
- `python3 scripts/check_secrets.py` — PASS
- Full `go test ./...` could not run in the sandbox because Go 1.25 toolchain download from `proxy.golang.org` is blocked by network/DNS restrictions. No Go/backend files were changed in this update.
