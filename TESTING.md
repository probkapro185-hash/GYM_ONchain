# Testing strategy

Проект содержит несколько уровней проверок. AI/RAG implementation в этой ревизии не менялся.

## 1. Unit/service tests

Проверяются основные бизнес-правила без PostgreSQL:

- auth/applications: нормализация, duplicate identities, login, inactive users, password checks;
- users/RBAC: client/manager/admin boundaries, password rotation, application transitions, deactivate rules;
- trainers: specialization, validation, create/update/deactivate;
- schedule: ownership/IDOR, overlaps, request approval/rejection, subscription reservation, completion, client cancellation window;
- finance/shop: exact cents, transactions, inactive products, subscription and sports purchases, soft delete;
- CRM+: progress, notes, tasks, notifications, freeze/unfreeze/extend, retention card aggregation;
- middleware: auth, stale token version, role guards, security headers, rate limits, request metrics;
- HTTP helpers and filters: strict JSON, size limits, IDs/dates, error mapping, finance query filters;
- PostgreSQL pure helpers: DB error mapping, RowsAffected checks, SQL payment filter construction;
- config/JWT/OpenAI client regression tests already present in the project.

Run:

```bash
make test
make test-shuffle
make test-race
```

`test-shuffle` helps reveal hidden ordering/global-state dependencies; `test-race` enables Go's race detector.

## 2. Fuzz tests

Validator fuzzing checks malformed/unexpected input and panic resistance:

```bash
make fuzz-smoke
```

Individual targets: `make fuzz-phone`, `make fuzz-money`, `make fuzz-name`.

## 3. Coverage gate

```bash
make coverage
make coverage-check
```

`COVERAGE_MIN` defaults to 70 and may be raised in CI:

```bash
make coverage-check COVERAGE_MIN=80
```

Coverage is a guardrail, not a substitute for business-rule assertions.

## 4. Frontend regression tests

```bash
make test-frontend
```

Checks include:

- JavaScript syntax via Node;
- duplicate/missing DOM IDs;
- critical frontend -> backend routes;
- no legacy self-topup endpoint;
- normal login/JWT integration without developer token fields;
- current RAG request contract unchanged;
- activation/reset/invite frontend contracts and absence of staff-entered client password;
- one-time token removal from browser URL and referrer protection;
- responsive/accessibility markers and CSS structure.

## 5. Static/security checks

```bash
make test-static
```

Checks migration sequence, common committed-secret patterns and Go formatting.

For production CI, also add a dedicated scanner such as Gitleaks/Trivy and dependency vulnerability scanning (`govulncheck`) when network access is available.

## 6. PostgreSQL integration smoke tests

Use a disposable, already migrated database:

```bash
export TEST_DATABASE_URL='postgres://...'
make integration
```

The integration suite checks required tables (including `account_tokens`), pgvector and clean/latest migration state. Never point it at a production database.

## 7. HTTP end-to-end smoke

With backend running:

```bash
make e2e
```

This checks `/healthz` and `/readyz`. For authenticated checks:

```bash
E2E_EMAIL='client@gmail.com' E2E_PASSWORD='...' make e2e
```

It then verifies login, `/users/me`, trainers and products. AI is intentionally not exercised by this new E2E script.

## Recommended CI order

```bash
make test-static
make test-frontend
go vet ./...
go test -shuffle=on -count=1 ./...
go test -race -count=1 ./...
make coverage-check COVERAGE_MIN=70
```

Run integration/E2E against disposable services in a separate CI job.
