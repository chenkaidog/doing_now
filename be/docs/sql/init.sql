DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT '0' COMMENT '删除时间戳(软删除)',
  `user_id` varchar(64) NOT NULL COMMENT '用户唯一标识ID',
  `account` varchar(64) NOT NULL COMMENT '用户登录账号',
  `name` varchar(64) NOT NULL COMMENT '用户姓名',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_user_id` (`user_id`),
  UNIQUE KEY `idx_users_account` (`account`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户表';

DROP TABLE IF EXISTS `user_credentials`;
CREATE TABLE `user_credentials` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT '0' COMMENT '删除时间戳(软删除)',
  `user_id` varchar(64) NOT NULL COMMENT '用户唯一标识ID',
  `password_salt` varchar(64) NOT NULL COMMENT '密码盐',
  `password_hash` varchar(128) NOT NULL COMMENT '密码哈希',
  `credential_version` int unsigned NOT NULL DEFAULT '0' COMMENT '密码凭证版本，每次修改密码的时候+1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户密码凭证表';

