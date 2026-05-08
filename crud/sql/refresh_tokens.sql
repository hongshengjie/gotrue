CREATE TABLE `refresh_tokens` (
  `id`          bigint(20)   NOT NULL AUTO_INCREMENT,
  `instance_id` varchar(255) NOT NULL DEFAULT '',
  `token`       varchar(255) NOT NULL DEFAULT '',
  `user_id`     varchar(255) NOT NULL DEFAULT '',
  `revoked`     tinyint(1)   NOT NULL DEFAULT 0,
  `created_at`  timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`  timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `refresh_tokens_instance_id_idx` (`instance_id`),
  KEY `refresh_tokens_instance_id_user_id_idx` (`instance_id`, `user_id`),
  KEY `refresh_tokens_token_idx` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
