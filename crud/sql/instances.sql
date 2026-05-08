CREATE TABLE `instances` (
  `id`             bigint(20)   NOT NULL AUTO_INCREMENT,
  `uid`            varchar(255) NOT NULL DEFAULT '',
  `uuid`           varchar(255) NOT NULL DEFAULT '',
  `raw_base_config` text        NOT NULL DEFAULT '',
  `created_at`     timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `instances_uid_uidx` (`uid`),
  UNIQUE KEY `instances_uuid_uidx` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
