# Infra assets

- **`init/`** — SQL run once when the Postgres data directory is first created (e.g. extensions).
- **Docker stack** — defined at the **repository root** in [`docker-compose.yml`](../docker-compose.yml). Use `docker compose up --build` or `make up-local` from the repo root.
