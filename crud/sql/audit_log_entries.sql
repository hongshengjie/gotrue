CREATE TABLE `audit_log_entries` (
  `id`          bigint(20)    NOT NULL AUTO_INCREMENT,
  `instance_id` int           NOT NULL DEFAULT 0,
  `payload`     varchar(4096) NOT NULL DEFAULT '{}',
  `created_at`  datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `audit_logs_instance_id_idx` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
