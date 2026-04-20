# Security Policy

Store `JWT_SECRET`, database credentials, and other secrets only in environment configuration (for example `.env` for local development); never commit real production values to the repository.

Password reset: in `ENVIRONMENT=local`, the API may return a raw `reset_token` for testing. In production, do **not** expose reset tokens in HTTP responses—deliver them only through your mailer or other private channel (see [devguide/DatabaseAndMigrations.md](devguide/DatabaseAndMigrations.md)).

## Reporting a Vulnerability

If you discover a security vulnerability, please open an issue.

### Preferred Languages

We prefer all communications in English.


Thank you 
