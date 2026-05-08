CREATE TABLE `audit_log_entries` (
  `id`          bigint(20)    NOT NULL AUTO_INCREMENT,
  `uid`         varchar(255)  NOT NULL DEFAULT '',
  `instance_id` varchar(255)  NOT NULL DEFAULT '',
  `payload`     varchar(4096) NOT NULL DEFAULT '{}',
  `created_at`  timestamp     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `audit_log_entries_uid_uidx` (`uid`),
  KEY `audit_logs_instance_id_idx` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
