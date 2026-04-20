# Local development

## Prerequisites

- Go 1.22+
- Docker (for PostgreSQL in `infra/`)

## Setup

1. Copy environment file:

   `cp .env.example .env`

2. If ports are already in use on your machine, edit `.env`:

   - **`DB_PORT`** / **`DB_FORWARD_PORT`** — host port mapped to Postgres (default `5432`). If another Postgres uses `5432`, use e.g. `5433` for both so the app and Docker use the same host port.
   - **`SERVER_PORT`** — HTTP port for the API (default `5000`).
   - **`ADMINER_PORT`** — Adminer UI (default `8080`).

3. Start the database (and Adminer):

   ```bash
   docker compose -f infra/docker-compose.yml --env-file .env up -d
   ```

4. Apply migrations:

   ```bash
   make migrate-apply
   ```

   The Makefile runs Atlas via `go run` (no global `atlas` install required). If `migrate apply` errors against your Postgres version, you can apply SQL manually:

   ```bash
   for f in migrations/*.sql; do
     docker exec -i multi-tenant-pg psql -U postgres -d app -v ON_ERROR_STOP=1 < "$f"
   done
   ```

   (Adjust container name / user / database if you changed them in `.env`.)

5. Run the API:

   ```bash
   go run . app:serve
   ```

   The command loads `.env` via `godotenv` and Viper. Environment variables set in your shell override values from `.env` when the same key is present (see `NewEnv` in `pkg/framework/env.go`).

## Auth endpoints (summary)

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/auth/register` | No |
| POST | `/api/auth/login` | No |
| POST | `/api/auth/tenant-session` | Bearer `pick_tenant_token` from login |
| POST | `/api/auth/refresh` | No |
| POST | `/api/auth/forgot-password` | No |
| POST | `/api/auth/reset-password` | No |
| GET | `/api/auth/me` | Bearer access token |
| PATCH | `/api/auth/me` | Bearer |
| POST | `/api/auth/change-password` | Bearer |
| **POST** | **`/api/auth/logout`** | **Bearer** |

### Login (two steps)

1. **POST `/api/auth/login`** with `email` and `password`. Response includes `user`, `tenants` (id, name, slug, role), and `pick_tenant_token` (short-lived; omitted or empty when the user has no memberships).
2. **POST `/api/auth/tenant-session`** with header `Authorization: Bearer <pick_tenant_token>` and JSON `{"tenant_id":"<uuid from tenants>"}`. Response is the usual `access_token` and `refresh_token` for that tenant.

Use the access token on protected routes; it is bound to the chosen tenant. The pick token is only for this exchange and is rejected by the normal JWT middleware.

### Logout

Revokes the given **refresh** token (idempotent).

```http
POST /api/auth/logout
Authorization: Bearer <access_token>
Content-Type: application/json

{"refresh_token":"<refresh_token_from_register_or_tenant-session>"}
```

Response: **204 No Content** on success.

## Tests

- **Unit tests** (SQLite in-memory for domain services, no DB required): `make test`
- **Integration tests** (PostgreSQL + `tenancy.WithTenant`): `TEST_DATABASE_URL='postgres://…' make test-integration`  
  Uses build tag `integration` (`domain/auth/auth_integration_test.go`, `domain/widget/widget_integration_test.go`). Prefer a dedicated database; tests run `AutoMigrate` on the models they use.

## Health and docs

- `GET /health-check`
- Swagger (non-production): `GET /swagger/index.html`

## GORM vs migrations

See [DatabaseAndMigrations.md](DatabaseAndMigrations.md).
