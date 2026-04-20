-include .env
export

# --- Docker (full stack: Postgres, migrations, API, Adminer) ---
up-local:
	@test -f .env || (cp .env.example .env && echo "Created .env from .env.example — edit secrets if needed.")
	$(MAKE) swagger
	@echo ""
	@echo "Browse (after containers are up / API is listening):"
	@set -a && . ./.env && set +a 2>/dev/null; \
	  echo "  Swagger  http://localhost:$${SERVER_PORT:-5000}/swagger/index.html"; \
	  echo "  Health   http://localhost:$${SERVER_PORT:-5000}/health-check"; \
	  echo "  Adminer  http://localhost:$${ADMINER_PORT:-8080}/"
	@echo ""
	docker compose up --build

up-local-detached:
	@test -f .env || (cp .env.example .env && echo "Created .env from .env.example — edit secrets if needed.")
	$(MAKE) swagger
	@echo ""
	@echo "Browse (API on host port from SERVER_PORT in .env):"
	@set -a && . ./.env && set +a 2>/dev/null; \
	  echo "  Swagger  http://localhost:$${SERVER_PORT:-5000}/swagger/index.html"; \
	  echo "  Health   http://localhost:$${SERVER_PORT:-5000}/health-check"; \
	  echo "  Adminer  http://localhost:$${ADMINER_PORT:-8080}/"
	@echo ""
	docker compose up --build -d

down-local:
	docker compose down

down-local-clean:
	docker compose down -v

# Atlas CLI must be recent (e.g. v0.14.2+). Install: https://atlasgo.sh
# Do not use `go run ariga.io/atlas/cmd/atlas@latest` — that Go module is stuck at v0.13.1 and fails on
# PostgreSQL with: postgres: unexpected number of rows: 1
ATLAS ?= atlas
# search_path=public: Atlas treats the DB as “clean”/scoped correctly; avoids “not clean: found schema public” on many setups.
PG_URL := postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable&search_path=public
MIGRATE_DIR := file://migrations

migrate-status:
	$(ATLAS) migrate status --dir "$(MIGRATE_DIR)" --url "$(PG_URL)"

migrate-diff:
	$(ATLAS) migrate diff --env gorm

migrate-apply:
	$(ATLAS) migrate apply --dir "$(MIGRATE_DIR)" --url "$(PG_URL)"

# Use when the database already matches all SQL through this version but has no atlas_schema_revisions history
# (e.g. migrations were applied manually with psql). Example: make migrate-baseline BASELINE=20260422120000
migrate-baseline:
	@test -n "$(BASELINE)" || (echo 'Set BASELINE to the migration version, e.g. BASELINE=20260422120000' && exit 1)
	$(ATLAS) migrate apply --dir "$(MIGRATE_DIR)" --url "$(PG_URL)" --baseline "$(BASELINE)"

migrate-down:
	$(ATLAS) migrate down --dir "$(MIGRATE_DIR)" --url "$(PG_URL)" --env gorm

migrate-hash:
	$(ATLAS) migrate hash --dir "$(MIGRATE_DIR)"

# Apply migrations via official Atlas image (use when you prefer not to install the CLI).
# Postgres on the host: set DB_HOST=host.docker.internal (macOS/Windows) or use --network host (Linux).
migrate-apply-docker:
	docker run --rm -v "$(CURDIR)":/src -w /src arigaio/atlas:latest migrate apply --dir "$(MIGRATE_DIR)" --url "$(PG_URL)"

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g main.go -o docs --parseDependency --parseInternal

lint-setup:
	python3 -m ensurepip --upgrade
	sudo pip3 install pre-commit
	pre-commit install
	pre-commit autoupdate

test:
	go test ./... -count=1

# Requires PostgreSQL. Set TEST_DATABASE_URL (e.g. postgres://user:pass@localhost:5432/dbname?sslmode=disable).
test-integration:
	go test -tags=integration ./... -count=1

.PHONY: up-local up-local-detached down-local down-local-clean migrate-status migrate-diff migrate-apply migrate-baseline migrate-down migrate-hash migrate-apply-docker swagger lint-setup test test-integration
