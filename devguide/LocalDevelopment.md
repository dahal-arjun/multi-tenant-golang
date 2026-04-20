# Local development

## Prerequisites

- **Docker** (recommended full stack) **or** Go 1.22+ with local PostgreSQL if you run the API on the host.

## One-command stack (recommended)

From the **repository root**:

1. Optional: `cp .env.example .env` and adjust ports/secrets (`make up-local` creates `.env` if it is missing).

2. Start **Postgres**, run **Atlas migrations**, **API** (live-mounted source), and **Adminer**:

   ```bash
   make up-local
   ```

   This runs **`make swagger`** first (regenerates `docs/` from handler comments), prints local URLs, then starts Compose. Equivalent to `docker compose up --build` plus that prep step.

   Run in the background: `make up-local-detached` (then `docker compose logs -f api`).

3. Open **Swagger**: `http://localhost:<SERVER_PORT>/swagger/index.html` (see `SERVER_PORT` in `.env`; the API listens on port **5000 inside the container**, mapped to `SERVER_PORT` on the host).

4. **Adminer**: `http://localhost:<ADMINER_PORT>` — use server **`postgres`**, database/user/password from `.env`.

5. Stop: `make down-local` · wipe DB volume: `make down-local-clean`

**Requires Docker Compose v2.10+** (for `service_completed_successfully` after migrations).

### If migrations fail (dirty / old volume)

```bash
make down-local-clean
make up-local
```

Or baseline an existing DB (host workflows): see [DatabaseAndMigrations.md](DatabaseAndMigrations.md) and the “Atlas” section below.

---

## Run API on the host (optional)

Use this when you prefer `go run` without the API container (you still need PostgreSQL).

1. `cp .env.example .env`

2. Ports: **`DB_PORT`** / **`DB_FORWARD_PORT`**, **`SERVER_PORT`**, **`ADMINER_PORT`** if defaults conflict.

3. Start **only** the database tools (no root `docker-compose` API stack):

   ```bash
   docker compose up -d postgres adminer
   ```

4. Install [Atlas CLI](https://atlasgo.io/getting-started#installation) (do not use `go run ariga.io/atlas/cmd/atlas@latest`; that module is stuck at v0.13.1 and often breaks on PostgreSQL).

5. `make migrate-apply` (or `make migrate-apply-docker` with `DB_HOST=host.docker.internal` when Postgres runs in Docker on macOS/Windows).

6. `go run . app:serve`

`docker compose` sets **`DB_HOST=postgres`** only for the **api** service in the full stack. On the host, keep **`DB_HOST=localhost`** (and `DB_PORT`/`DB_FORWARD_PORT` aligned) in `.env`.

Manual SQL fallback:

```bash
for f in migrations/*.sql; do
  docker exec -i multi-tenant-pg psql -U postgres -d app -v ON_ERROR_STOP=1 < "$f"
done
```

(Adjust container name / user / database from `.env`.)

**If Atlas says** `connected database is not clean…`:

- Wipe volume: `docker compose down -v`, then `docker compose up -d postgres`, then `make migrate-apply`.
- Or `make migrate-baseline BASELINE=20260424120000` if the schema already matches all migrations (adjust version as needed).

---

## Auth endpoints (summary)

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/auth/register` | No (bootstrap: verified user + tenant + tokens) |
| POST | `/api/auth/signup` | No (self-serve: user only; verify email next) |
| POST | `/api/auth/verify-email` | No |
| POST | `/api/auth/resend-verification` | No (rate-limited: 3/hour per email+IP) |
| POST | `/api/auth/login` | No |
| POST | `/api/auth/tenant-session` | Bearer `pick_tenant_token` from login |
| POST | `/api/auth/create-tenant` | Bearer `pick_tenant_token` (first org when `tenants` is empty) |
| POST | `/api/auth/accept-invite` | No |
| POST | `/api/auth/refresh` | No |
| POST | `/api/auth/forgot-password` | No |
| POST | `/api/auth/reset-password` | No |
| GET | `/api/auth/me` | Bearer access token |
| PATCH | `/api/auth/me` | Bearer |
| POST | `/api/auth/change-password` | Bearer |
| POST | `/api/auth/logout` | Bearer |
| POST | `/api/invites` | Bearer access token (tenant **owner** or **admin** only) |
| GET/POST/PATCH/DELETE | `/api/tenant-roles`, `/api/tenant-roles/:id`, … | Bearer tenant access (see below) |
| PATCH | `/api/tenant-members/role` | Bearer (assign custom `tenant_role_id` to a member) |
| GET | `/api/platform/health`, `/api/platform/tenants/summary` | Bearer **platform** access token (see below) |

### Self-serve onboarding (verify → first tenant)

1. **POST `/api/auth/signup`** with `email` and `password`. In `ENVIRONMENT=local`, the JSON may include `verification_token` for testing.
2. **POST `/api/auth/verify-email`** with `{"token":"..."}` (or open `PUBLIC_APP_URL` + `EMAIL_VERIFICATION_PATH` with that token in your frontend).
3. **POST `/api/auth/login`**. Until the email is verified, login returns **403** with a distinct error: within the grace window (`EMAIL_VERIFICATION_TTL_MINUTES` from signup or last resend), use verify/resend; after the window, the account is deactivated until **POST `/api/auth/resend-verification`** extends the window and re-enables the account path.
4. If `tenants` is empty, the response still includes **`pick_tenant_token`**. Call **POST `/api/auth/create-tenant`** with `Authorization: Bearer <pick_tenant_token>` and `{"tenant_name":"..."}` to create the organization and receive `access_token` / `refresh_token`.
5. If the user already has tenants, use **`tenant-session`** as before.

### Login (existing members)

1. **POST `/api/auth/login`** with `email` and `password`. Response includes `user`, `tenants`, and `pick_tenant_token` (short-lived). If `users.role` is **`admin`** or **`system_manager`**, the response may also include **`platform_access_token`**, **`platform_refresh_token`**, and **`platform_expires_in`** for **platform** APIs (no tenant in JWT).
2. **POST `/api/auth/tenant-session`** with header `Authorization: Bearer <pick_tenant_token>` and JSON `{"tenant_id":"<uuid from tenants>"}`. Response is the usual `access_token` and `refresh_token` for that tenant.

Use the tenant **access** token on tenant-scoped routes; it includes `tid`, legacy `trole`, optional `trid` (custom tenant role id), and `perms` (effective permission keys). The pick token is only for `tenant-session` and `create-tenant`. **Refresh**: use the matching refresh token; tenant refreshes include a tenant id; platform refreshes omit it—`POST /api/auth/refresh` returns the same kind of pair you sent.

### Tenant invitations

- **POST `/api/invites`** with Bearer access token: body `{"email":"...","role":"member|admin|owner","tenant_role_id":"<optional uuid>"}` (custom role must belong to the current tenant). In `ENVIRONMENT=local`, `invite_token` may be returned for testing. New invitees complete **POST `/api/auth/accept-invite`** with `token` and `password` (existing users must send their **current** password).

### Curated permissions and custom tenant roles

- Permission keys are defined in code ([`domain/constants/permissions.go`](domain/constants/permissions.go)), e.g. `widgets.read`, `widgets.write`, `roles.manage`, `invites.manage`, `members.manage`, `platform.read`.
- Built-in **owner** / **admin** / **member** still exist on each membership; effective JWT permissions are **builtin(role) ∪ grants(custom role)** when `tenant_role_id` is set on the membership.
- Tenant admins (built-in owner/admin, or anyone with `roles.manage`) can manage **GET/POST `/api/tenant-roles`**, **PATCH/DELETE `/api/tenant-roles/:id`**, **PUT `/api/tenant-roles/:id/permissions`**, and **PATCH `/api/tenant-members/role`**.
- Example: **GET `/api/widgets`** requires **`widgets.read`** on the access token.

### Platform (system) operators

- Seeded admin user uses **`system_manager`** ([`seeds/admin_seed.go`](seeds/admin_seed.go)); **`admin`** is also allowed for platform JWTs.
- Call **`GET /api/platform/tenants/summary`** with `Authorization: Bearer <platform_access_token>`. These handlers **do not** set PostgreSQL RLS `app.current_tenant_id`; they run as a normal DB session—only use for trusted operators.

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
