-- Create "users" table
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `uuid` binary(16) NOT NULL,
  `cognito_uid` varchar(50) NULL,
  `first_name` varchar(255) NULL,
  `last_name` varchar(255) NULL,
  `first_name_ja` varchar(255) NULL,
  `last_name_ja` varchar(255) NULL,
  `email` varchar(255) NOT NULL,
  `role` varchar(25) NULL,
  `is_active` bool NULL DEFAULT 0,
  `is_email_verified` bool NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  INDEX `idx_users_cognito_uid` (`cognito_uid`),
  INDEX `idx_users_deleted_at` (`deleted_at`),
  INDEX `idx_users_uuid` (`uuid`),
  UNIQUE INDEX `uni_users_cognito_uid` (`cognito_uid`),
  UNIQUE INDEX `uni_users_uuid` (`uuid`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
