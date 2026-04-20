# Database: GORM and SQL migrations

This project uses **both** GORM and **versioned SQL migrations**. They solve different problems.

## GORM (runtime)

- **Models** live in [`domain/models`](../domain/models). They describe how Go maps table rows and drive queries via [`pkg/infrastructure/db.go`](../pkg/infrastructure/db.go) (`*gorm.DB`).
- Repositories and services use GORM for `Create`, `Find`, `Where`, transactions, etc.

GORM is the **application’s ORM**, not the single source of truth for every database feature.

## SQL migrations (schema history)

- Files under [`migrations/`](../migrations/) are applied in order (Atlas: `make migrate-apply`).
- They define **DDL** that must be explicit and reviewable: tables, indexes, foreign keys, and **PostgreSQL Row Level Security (RLS)** policies.

GORM’s `AutoMigrate` does **not** reliably express or maintain RLS, extensions, partial indexes, and many Postgres-specific details. Keeping schema changes in SQL gives:

- The same steps on every environment (dev, CI, production).
- PR-friendly diffs and optional rollbacks (`make migrate-down`).
- A clear audit trail of how the database evolved.

## Atlas

Install a **current** Atlas CLI ([install guide](https://atlasgo.io/getting-started#installation)). The Go module `ariga.io/atlas/cmd/atlas` is no longer updated on the proxy (last tag v0.13.1); `go run …/cmd/atlas@latest` commonly fails against PostgreSQL with `postgres: unexpected number of rows: 1`. The project `Makefile` invokes the `atlas` binary (`ATLAS` overrideable); use `make migrate-apply-docker` if you prefer the official container.

If `migrate apply` reports the database is **not clean** (often after manual `psql` applies or a reused volume without `atlas_schema_revisions`), see **LocalDevelopment.md** (`migrate-baseline` vs `docker compose down -v`).

[`atlas.hcl`](../atlas.hcl) wires the **Atlas GORM provider** so you can run `make migrate-diff` to **propose** new SQL from model changes. You still **review and edit** the generated SQL—especially for RLS and policies—then commit it and run `make migrate-hash` when files change.

## Practical rule

| Concern | Where it lives |
|--------|----------------|
| Column types, associations, queries | GORM models + code |
| Table creation, RLS, extensions, precise FK behavior | SQL migrations |

For tenant-scoped queries at runtime, see [`pkg/tenancy`](../pkg/tenancy) and [`pkg/dbscope`](../pkg/dbscope); for reusable columns, see [`domain/models/mixins.go`](../domain/models/mixins.go).

## Password reset (no email in base template)

Forgot-password stores a **hashed token** in `password_reset_tokens`. In **`ENVIRONMENT=local`**, the API may return the raw `reset_token` in the JSON response so you can test without mail. In **production**, integrate email (or another channel) and **do not** return the token in the HTTP body—send it only to the user through your delivery mechanism.
