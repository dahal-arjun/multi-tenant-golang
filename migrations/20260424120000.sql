-- atlas:txmode file

CREATE TABLE "tenant_roles" (
  "id" uuid NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "tenant_id" uuid NOT NULL,
  "name" varchar(255) NOT NULL,
  "slug" varchar(100) NOT NULL,
  "description" varchar(500) NULL,
  "created_by_id" bigint NULL,
  "updated_by_id" bigint NULL,
  "deleted_by_id" bigint NULL,
  CONSTRAINT "fk_tenant_roles_tenant" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id") ON DELETE CASCADE,
  CONSTRAINT "fk_tenant_roles_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "fk_tenant_roles_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "fk_tenant_roles_deleted_by" FOREIGN KEY ("deleted_by_id") REFERENCES "users" ("id") ON DELETE SET NULL
);
CREATE INDEX "idx_tenant_roles_tenant_id" ON "tenant_roles" ("tenant_id");
CREATE INDEX "idx_tenant_roles_deleted_at" ON "tenant_roles" ("deleted_at");
CREATE UNIQUE INDEX "idx_tenant_roles_tenant_slug" ON "tenant_roles" ("tenant_id", "slug") WHERE "deleted_at" IS NULL;

CREATE TABLE "tenant_role_permissions" (
  "id" bigserial NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "tenant_role_id" uuid NOT NULL,
  "permission_key" varchar(100) NOT NULL,
  CONSTRAINT "fk_tenant_role_permissions_role" FOREIGN KEY ("tenant_role_id") REFERENCES "tenant_roles" ("id") ON DELETE CASCADE,
  CONSTRAINT "uni_tenant_role_permissions_role_key" UNIQUE ("tenant_role_id", "permission_key")
);
CREATE INDEX "idx_tenant_role_permissions_tenant_role_id" ON "tenant_role_permissions" ("tenant_role_id");

ALTER TABLE "tenant_memberships" ADD COLUMN "tenant_role_id" uuid NULL;
ALTER TABLE "tenant_memberships" ADD CONSTRAINT "fk_tenant_memberships_tenant_role" FOREIGN KEY ("tenant_role_id") REFERENCES "tenant_roles" ("id") ON DELETE SET NULL;
CREATE INDEX "idx_tenant_memberships_tenant_role_id" ON "tenant_memberships" ("tenant_role_id");

ALTER TABLE "tenant_invitations" ADD COLUMN "tenant_role_id" uuid NULL;
ALTER TABLE "tenant_invitations" ADD CONSTRAINT "fk_tenant_invitations_tenant_role" FOREIGN KEY ("tenant_role_id") REFERENCES "tenant_roles" ("id") ON DELETE SET NULL;

ALTER TABLE "refresh_tokens" ALTER COLUMN "tenant_id" DROP NOT NULL;
