-- atlas:txmode read-write

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE "users" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "uuid" uuid NOT NULL,
  "password_hash" varchar(255) NOT NULL,
  "first_name" varchar(255) NULL,
  "last_name" varchar(255) NULL,
  "first_name_ja" varchar(255) NULL,
  "last_name_ja" varchar(255) NULL,
  "email" varchar(255) NOT NULL,
  "role" varchar(25) NULL,
  "is_active" bool DEFAULT false,
  "is_email_verified" bool DEFAULT false
);
CREATE UNIQUE INDEX "uni_users_uuid" ON "users" ("uuid");
CREATE UNIQUE INDEX "uni_users_email" ON "users" ("email");
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");

CREATE TABLE "tenants" (
  "id" uuid NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "name" varchar(255) NOT NULL,
  "slug" varchar(255) NOT NULL
);
CREATE UNIQUE INDEX "uni_tenants_slug" ON "tenants" ("slug");
CREATE INDEX "idx_tenants_deleted_at" ON "tenants" ("deleted_at");

CREATE TABLE "tenant_memberships" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "user_id" bigint NOT NULL,
  "tenant_id" uuid NOT NULL,
  "role" varchar(50) NOT NULL,
  CONSTRAINT "fk_tenant_memberships_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id"),
  CONSTRAINT "fk_tenant_memberships_tenant" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id")
);
CREATE INDEX "idx_tenant_memberships_user_id" ON "tenant_memberships" ("user_id");
CREATE INDEX "idx_tenant_memberships_tenant_id" ON "tenant_memberships" ("tenant_id");
CREATE INDEX "idx_tenant_memberships_deleted_at" ON "tenant_memberships" ("deleted_at");
CREATE UNIQUE INDEX "idx_tenant_memberships_active_user_tenant" ON "tenant_memberships" ("user_id", "tenant_id") WHERE "deleted_at" IS NULL;

CREATE TABLE "refresh_tokens" (
  "id" uuid NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "user_id" bigint NOT NULL,
  "tenant_id" uuid NOT NULL,
  "token_hash" varchar(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "revoked_at" timestamptz NULL,
  CONSTRAINT "fk_refresh_tokens_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id"),
  CONSTRAINT "fk_refresh_tokens_tenant" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id")
);
CREATE INDEX "idx_refresh_tokens_user_id" ON "refresh_tokens" ("user_id");
CREATE INDEX "idx_refresh_tokens_deleted_at" ON "refresh_tokens" ("deleted_at");

CREATE TABLE "widgets" (
  "id" uuid NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "tenant_id" uuid NOT NULL,
  "title" varchar(255) NOT NULL,
  CONSTRAINT "fk_widgets_tenant" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id")
);
CREATE INDEX "idx_widgets_tenant_id" ON "widgets" ("tenant_id");
CREATE INDEX "idx_widgets_deleted_at" ON "widgets" ("deleted_at");

ALTER TABLE "widgets" ENABLE ROW LEVEL SECURITY;
ALTER TABLE "widgets" FORCE ROW LEVEL SECURITY;
CREATE POLICY "widgets_tenant_isolation" ON "widgets" AS PERMISSIVE FOR ALL TO PUBLIC
  USING ("tenant_id" = current_setting('app.current_tenant_id', true)::uuid)
  WITH CHECK ("tenant_id" = current_setting('app.current_tenant_id', true)::uuid);
