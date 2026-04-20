-- atlas:txmode file

ALTER TABLE "users" ADD COLUMN "email_verification_deadline" timestamptz NULL;

CREATE TABLE "email_verification_tokens" (
  "id" uuid NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "user_id" bigint NOT NULL,
  "token_hash" varchar(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz NULL,
  CONSTRAINT "fk_email_verification_tokens_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE
);
CREATE INDEX "idx_email_verification_tokens_user_id" ON "email_verification_tokens" ("user_id");
CREATE INDEX "idx_email_verification_tokens_token_hash" ON "email_verification_tokens" ("token_hash");

CREATE TABLE "tenant_invitations" (
  "id" uuid NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "tenant_id" uuid NOT NULL,
  "email" varchar(255) NOT NULL,
  "role" varchar(50) NOT NULL,
  "token_hash" varchar(64) NOT NULL,
  "invited_by_user_id" bigint NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "accepted_at" timestamptz NULL,
  "revoked_at" timestamptz NULL,
  CONSTRAINT "fk_tenant_invitations_tenant" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id") ON DELETE CASCADE,
  CONSTRAINT "fk_tenant_invitations_invited_by" FOREIGN KEY ("invited_by_user_id") REFERENCES "users" ("id") ON DELETE CASCADE
);
CREATE INDEX "idx_tenant_invitations_tenant_id" ON "tenant_invitations" ("tenant_id");
CREATE INDEX "idx_tenant_invitations_token_hash" ON "tenant_invitations" ("token_hash");
CREATE UNIQUE INDEX "idx_tenant_invitations_pending_tenant_email" ON "tenant_invitations" ("tenant_id", "email") WHERE "accepted_at" IS NULL AND "revoked_at" IS NULL;
