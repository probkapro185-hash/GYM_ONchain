.PHONY: run fmt fmt-check vet test test-shuffle test-race test-frontend test-static test-all \
        coverage coverage-check fuzz-phone fuzz-money fuzz-name fuzz-smoke integration e2e \
        check docker-up docker-down docker-logs

GO ?= go
PYTHON ?= python3
COVERAGE_MIN ?= 70

run:
	$(GO) run ./cmd/server

fmt:
	find cmd internal pkg -name '*.go' -print0 | xargs -0 gofmt -w

fmt-check:
	@files="$$(find cmd internal pkg integration -name '*.go' -type f)"; out="$$(gofmt -l $$files)"; test -z "$$out" || { echo 'Go files need gofmt:'; echo "$$out"; exit 1; }

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

test-shuffle:
	$(GO) test -shuffle=on -count=1 ./...

test-race:
	$(GO) test -race -count=1 ./...

test-frontend:
	$(PYTHON) scripts/verify_frontend.py
	$(PYTHON) scripts/test_frontend_contract.py
	node --check frontend/assets/app.js
	node --test scripts/test_frontend_behavior.cjs

test-static:
	$(PYTHON) scripts/check_migrations.py
	$(PYTHON) scripts/check_secrets.py
	$(MAKE) fmt-check

test-all: test-static test-frontend vet test-shuffle

coverage:
	$(GO) test -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tee coverage.txt

coverage-check: coverage
	$(PYTHON) scripts/check_coverage.py coverage.txt $(COVERAGE_MIN)

fuzz-phone:
	$(GO) test ./internal/validator -run='^$$' -fuzz=FuzzNormalizePhoneNeverPanics -fuzztime=10s

fuzz-money:
	$(GO) test ./internal/validator -run='^$$' -fuzz=FuzzMoneyToCentsRoundTrip -fuzztime=10s

fuzz-name:
	$(GO) test ./internal/validator -run='^$$' -fuzz=FuzzValidateFullNameNeverPanics -fuzztime=10s

fuzz-smoke: fuzz-phone fuzz-money fuzz-name

# Requires TEST_DATABASE_URL pointing at an already migrated disposable database.
integration:
	@test -n "$$TEST_DATABASE_URL" || { echo 'Set TEST_DATABASE_URL first'; exit 2; }
	$(GO) test -tags=integration -count=1 ./integration/...

# With no credentials this checks only health/readiness. Set E2E_EMAIL/E2E_PASSWORD for auth smoke tests.
e2e:
	bash scripts/e2e_smoke.sh

check: test-all

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f app
