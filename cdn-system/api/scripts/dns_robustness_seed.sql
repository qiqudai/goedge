SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS `region` (
  `id` bigint NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `des` varchar(255) DEFAULT NULL,
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `user` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `email` varchar(191) DEFAULT NULL,
  `name` varchar(191) DEFAULT NULL,
  `des` longtext,
  `phone` longtext,
  `qq` longtext,
  `cert_id` longtext,
  `cert_name` longtext,
  `cert_no` longtext,
  `cert_verified` tinyint(1) DEFAULT NULL,
  `white_ip` longtext,
  `login_captcha` longtext,
  `balance` bigint DEFAULT NULL,
  `freeze` bigint DEFAULT NULL,
  `create_at` datetime(3) DEFAULT NULL,
  `password` longtext,
  `enable` tinyint(1) DEFAULT NULL,
  `type` bigint DEFAULT NULL,
  `company` longtext,
  `tea_code` longtext,
  `secondary_auth` tinyint(1) DEFAULT NULL,
  `secondary_auth_deadline` longtext,
  `secondary_auth_action` longtext,
  `secondary_auth_status` longtext,
  `group_id` bigint DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `dnsapi` (
  `id` bigint NOT NULL,
  `uid` int DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `des` varchar(255) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `auth` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `cname_domains` (
  `id` bigint NOT NULL,
  `domain` varchar(255) NOT NULL,
  `note` varchar(255) DEFAULT '',
  `create_at` datetime DEFAULT NULL,
  `update_at` datetime DEFAULT NULL,
  `dns_provider_id` bigint DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `region` (`id`, `name`, `create_at`, `update_at`)
VALUES (1, 'dns-e2e-region', NOW(), NOW())
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

INSERT INTO `dnsapi` (`id`, `uid`, `name`, `des`, `type`, `auth`)
VALUES (1, 0, 'memory-dns', 'dns robustness e2e', 'memory', '{}')
ON DUPLICATE KEY UPDATE `type` = VALUES(`type`), `auth` = VALUES(`auth`);

INSERT INTO `cname_domains` (`id`, `domain`, `note`, `create_at`, `update_at`, `dns_provider_id`)
VALUES (1, 'dns-e2e.local', 'robustness test only', NOW(), NOW(), 1)
ON DUPLICATE KEY UPDATE `dns_provider_id` = VALUES(`dns_provider_id`);

UPDATE `node_group`
SET `cname_hostname` = 'linea01',
    `cname_domain` = 'dns-e2e.local',
    `update_at` = NOW()
WHERE `id` = 1;

UPDATE `node`
SET `region_id` = 1,
    `enable` = 1,
    `update_at` = NOW(3)
WHERE `id` = 1;

INSERT INTO `node` (`id`, `name`, `ip`, `enable`, `region_id`, `level`, `create_at`, `update_at`)
VALUES
  (2, 'dns-e2e-node-2', '203.0.113.2', 1, 1, 1, NOW(3), NOW(3)),
  (3, 'dns-e2e-node-3', '203.0.113.3', 1, 1, 1, NOW(3), NOW(3)),
  (4, 'dns-e2e-node-4', '203.0.113.4', 1, 1, 1, NOW(3), NOW(3)),
  (5, 'dns-e2e-node-5', '203.0.113.5', 1, 1, 1, NOW(3), NOW(3)),
  (6, 'dns-e2e-node-6', '203.0.113.6', 1, 1, 1, NOW(3), NOW(3)),
  (7, 'dns-e2e-node-7', '203.0.113.7', 1, 1, 1, NOW(3), NOW(3)),
  (8, 'dns-e2e-node-8', '203.0.113.8', 1, 1, 1, NOW(3), NOW(3)),
  (9, 'dns-e2e-node-9', '203.0.113.9', 1, 1, 1, NOW(3), NOW(3)),
  (10, 'dns-e2e-node-10', '203.0.113.10', 1, 1, 1, NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
  `ip` = VALUES(`ip`),
  `enable` = VALUES(`enable`),
  `region_id` = VALUES(`region_id`),
  `update_at` = VALUES(`update_at`);

UPDATE `line`
SET `node_ip_id` = 1,
    `enable` = 1,
    `update_at` = NOW()
WHERE `id` = 1;

INSERT INTO `line` (
  `id`, `node_group_id`, `node_id`, `node_ip_id`, `line_id`, `line_name`, `weight`,
  `create_at`, `update_at`, `enable`
)
SELECT n.id + 100, 1, n.id, n.id, 'default', 'default', '1', NOW(), NOW(), 1
FROM `node` n
WHERE n.id BETWEEN 2 AND 10
  AND NOT EXISTS (
    SELECT 1 FROM `line` l
    WHERE l.node_group_id = 1 AND l.node_ip_id = n.id AND l.line_id = 'default'
  );

UPDATE `node`
SET `ip` = '203.0.113.1',
    `region_id` = 1,
    `enable` = 1
WHERE `id` = 1;

ALTER TABLE `line` MODIFY `id` int NOT NULL AUTO_INCREMENT;
