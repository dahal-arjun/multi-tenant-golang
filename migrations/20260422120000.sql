-- atlas:txmode read-write

CREATE TABLE "password_reset_tokens" (
  "id" uuid NOT NULL PRIMARY KEY,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "user_id" bigint NOT NULL,
  "token_hash" varchar(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz NULL,
  CONSTRAINT "fk_password_reset_tokens_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE
);
CREATE INDEX "idx_password_reset_tokens_user_id" ON "password_reset_tokens" ("user_id");
CREATE INDEX "idx_password_reset_tokens_token_hash" ON "password_reset_tokens" ("token_hash");
