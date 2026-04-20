-- atlas:txmode read-write

ALTER TABLE "users" ADD COLUMN "tenant_id" uuid NULL;
ALTER TABLE "users" ADD COLUMN "created_by_id" bigint NULL;
ALTER TABLE "users" ADD COLUMN "updated_by_id" bigint NULL;
ALTER TABLE "users" ADD COLUMN "deleted_by_id" bigint NULL;
ALTER TABLE "users" ADD CONSTRAINT "fk_users_tenant" FOREIGN KEY ("tenant_id") REFERENCES "tenants" ("id") ON DELETE SET NULL;
ALTER TABLE "users" ADD CONSTRAINT "fk_users_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "users" ADD CONSTRAINT "fk_users_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "users" ADD CONSTRAINT "fk_users_deleted_by" FOREIGN KEY ("deleted_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
CREATE INDEX "idx_users_tenant_id" ON "users" ("tenant_id");
CREATE INDEX "idx_users_created_by_id" ON "users" ("created_by_id");

ALTER TABLE "tenants" ADD COLUMN "created_by_id" bigint NULL;
ALTER TABLE "tenants" ADD COLUMN "updated_by_id" bigint NULL;
ALTER TABLE "tenants" ADD COLUMN "deleted_by_id" bigint NULL;
ALTER TABLE "tenants" ADD CONSTRAINT "fk_tenants_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "tenants" ADD CONSTRAINT "fk_tenants_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "tenants" ADD CONSTRAINT "fk_tenants_deleted_by" FOREIGN KEY ("deleted_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;

ALTER TABLE "tenant_memberships" ADD COLUMN "created_by_id" bigint NULL;
ALTER TABLE "tenant_memberships" ADD COLUMN "updated_by_id" bigint NULL;
ALTER TABLE "tenant_memberships" ADD COLUMN "deleted_by_id" bigint NULL;
ALTER TABLE "tenant_memberships" ADD CONSTRAINT "fk_tenant_memberships_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "tenant_memberships" ADD CONSTRAINT "fk_tenant_memberships_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "tenant_memberships" ADD CONSTRAINT "fk_tenant_memberships_deleted_by" FOREIGN KEY ("deleted_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;

ALTER TABLE "refresh_tokens" ADD COLUMN "created_by_id" bigint NULL;
ALTER TABLE "refresh_tokens" ADD COLUMN "updated_by_id" bigint NULL;
ALTER TABLE "refresh_tokens" ADD COLUMN "deleted_by_id" bigint NULL;
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "fk_refresh_tokens_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "fk_refresh_tokens_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "fk_refresh_tokens_deleted_by" FOREIGN KEY ("deleted_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;

ALTER TABLE "widgets" ADD COLUMN "created_by_id" bigint NULL;
ALTER TABLE "widgets" ADD COLUMN "updated_by_id" bigint NULL;
ALTER TABLE "widgets" ADD COLUMN "deleted_by_id" bigint NULL;
ALTER TABLE "widgets" ADD CONSTRAINT "fk_widgets_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "widgets" ADD CONSTRAINT "fk_widgets_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "widgets" ADD CONSTRAINT "fk_widgets_deleted_by" FOREIGN KEY ("deleted_by_id") REFERENCES "users" ("id") ON DELETE SET NULL;
