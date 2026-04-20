#!/usr/bin/env bash
set -euo pipefail

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-123456}"
MYSQL_DB="${MYSQL_DB:-cdnfy}"

MYSQL_BASE_ARGS=(-h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" "-p${MYSQL_PASSWORD}" --default-character-set=utf8mb4)

mysql "${MYSQL_BASE_ARGS[@]}" -e "CREATE DATABASE IF NOT EXISTS \`${MYSQL_DB}\` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;"

mysql "${MYSQL_BASE_ARGS[@]}" "${MYSQL_DB}" <<'SQL'
SET @now := NOW();

SET @region_id := (SELECT id FROM region ORDER BY id LIMIT 1);
INSERT INTO region (name, des, create_at, update_at)
SELECT 'local-region', 'local bootstrap', @now, @now
WHERE @region_id IS NULL;
SET @region_id := COALESCE(@region_id, LAST_INSERT_ID());

SET @node_group_id := (SELECT id FROM node_group ORDER BY id LIMIT 1);
INSERT INTO node_group (region_id, cname_hostname, cname_domain, name, des, create_at, update_at)
SELECT @region_id, 'edge', 'cnn-ai.test', 'local-group', 'local bootstrap', @now, @now
WHERE @node_group_id IS NULL;
SET @node_group_id := COALESCE(@node_group_id, LAST_INSERT_ID());

SET @package_id := (SELECT id FROM package ORDER BY id LIMIT 1);
INSERT INTO package (name, des, region_id, node_group_id, cname_domain, cname_mode, domain, enable, create_at, update_at)
SELECT 'local-package', 'local bootstrap', @region_id, @node_group_id, 'cnn-ai.test', 'site', 100, 1, @now, @now
WHERE @package_id IS NULL;
SET @package_id := COALESCE(@package_id, LAST_INSERT_ID());

SET @admin_id := (SELECT id FROM user WHERE name = 'cnn_ai_admin' ORDER BY id LIMIT 1);
INSERT INTO user (email, name, des, phone, qq, cert_id, cert_name, cert_no, cert_verified, white_ip, login_captcha, balance, `freeze`, create_at, `password`, `enable`, `type`)
SELECT 'cnn_ai_admin@local.test', 'cnn_ai_admin', '', '', '', '', '', '', 0, '', '', 0, 0, @now, 'admin123', 1, 1
WHERE @admin_id IS NULL;
SET @admin_id := COALESCE(@admin_id, LAST_INSERT_ID());
UPDATE user
SET email='cnn_ai_admin@local.test',
    des='',
    phone='',
    qq='',
    cert_id='',
    cert_name='',
    cert_no='',
    cert_verified=0,
    white_ip='',
    login_captcha='',
    balance=0,
    `freeze`=0,
    `password`='admin123',
    `enable`=1,
    `type`=1
WHERE id=@admin_id;

SET @user_id := (SELECT id FROM user WHERE name = 'cnn_ai_user' ORDER BY id LIMIT 1);
INSERT INTO user (email, name, des, phone, qq, cert_id, cert_name, cert_no, cert_verified, white_ip, login_captcha, balance, `freeze`, create_at, `password`, `enable`, `type`)
SELECT 'cnn_ai_user@local.test', 'cnn_ai_user', '', '', '', '', '', '', 0, '', '', 0, 0, @now, 'user123', 1, 0
WHERE @user_id IS NULL;
SET @user_id := COALESCE(@user_id, LAST_INSERT_ID());
UPDATE user
SET email='cnn_ai_user@local.test',
    des='',
    phone='',
    qq='',
    cert_id='',
    cert_name='',
    cert_no='',
    cert_verified=0,
    white_ip='',
    login_captcha='',
    balance=0,
    `freeze`=0,
    `password`='user123',
    `enable`=1,
    `type`=0
WHERE id=@user_id;

SET @dns_id := (SELECT id FROM dnsapi WHERE uid=@admin_id ORDER BY id LIMIT 1);
INSERT INTO dnsapi (uid, name, des, `type`, auth)
SELECT @admin_id, 'local-dns', 'local bootstrap', 'huawei', '{}'
WHERE @dns_id IS NULL;
SET @dns_id := COALESCE(@dns_id, LAST_INSERT_ID());
UPDATE dnsapi
SET name='local-dns',
    des='local bootstrap',
    `type`='huawei',
    auth='{}'
WHERE id=@dns_id;

SET @admin_pkg_id := (SELECT id FROM user_package WHERE uid=@admin_id ORDER BY id LIMIT 1);
INSERT INTO user_package (
    uid, name, `package`, region_id, node_group_id, backup_node_group, enable_backup_group,
    cname_domain, cname_mode, traffic, bandwidth, connection, domain, main_domain_limit,
    http_port, stream_port, custom_cc_rule, websocket, month_price, quarter_price, year_price,
    create_at, start_at, end_at, version, is_expired, l2_origin
)
SELECT
    @admin_id, 'admin-local-package', @package_id, @region_id, @node_group_id, NULL, 0,
    'cnn-ai.test', 'site', 102400, '100', 10000, 100, 100,
    80, 0, 1, 1, 0, 0, 0,
    @now, @now, DATE_ADD(@now, INTERVAL 365 DAY), 1, 0, 0
WHERE @admin_pkg_id IS NULL;
SET @admin_pkg_id := COALESCE(@admin_pkg_id, LAST_INSERT_ID());
UPDATE user_package
SET name='admin-local-package',
    `package`=@package_id,
    region_id=@region_id,
    node_group_id=@node_group_id,
    enable_backup_group=0,
    cname_domain='cnn-ai.test',
    cname_mode='site',
    traffic=102400,
    bandwidth='100',
    connection=10000,
    domain=100,
    main_domain_limit=100,
    http_port=80,
    stream_port=0,
    custom_cc_rule=1,
    websocket=1,
    month_price=0,
    quarter_price=0,
    year_price=0,
    start_at=@now,
    end_at=DATE_ADD(@now, INTERVAL 365 DAY),
    version=1,
    is_expired=0,
    l2_origin=0
WHERE id=@admin_pkg_id;

SET @user_pkg_id := (SELECT id FROM user_package WHERE uid=@user_id ORDER BY id LIMIT 1);
INSERT INTO user_package (
    uid, name, `package`, region_id, node_group_id, backup_node_group, enable_backup_group,
    cname_domain, cname_mode, traffic, bandwidth, connection, domain, main_domain_limit,
    http_port, stream_port, custom_cc_rule, websocket, month_price, quarter_price, year_price,
    create_at, start_at, end_at, version, is_expired, l2_origin
)
SELECT
    @user_id, 'user-local-package', @package_id, @region_id, @node_group_id, NULL, 0,
    'cnn-ai.test', 'site', 102400, '100', 10000, 100, 100,
    80, 0, 1, 1, 0, 0, 0,
    @now, @now, DATE_ADD(@now, INTERVAL 365 DAY), 1, 0, 0
WHERE @user_pkg_id IS NULL;
SET @user_pkg_id := COALESCE(@user_pkg_id, LAST_INSERT_ID());
UPDATE user_package
SET name='user-local-package',
    `package`=@package_id,
    region_id=@region_id,
    node_group_id=@node_group_id,
    enable_backup_group=0,
    cname_domain='cnn-ai.test',
    cname_mode='site',
    traffic=102400,
    bandwidth='100',
    connection=10000,
    domain=100,
    main_domain_limit=100,
    http_port=80,
    stream_port=0,
    custom_cc_rule=1,
    websocket=1,
    month_price=0,
    quarter_price=0,
    year_price=0,
    start_at=@now,
    end_at=DATE_ADD(@now, INTERVAL 365 DAY),
    version=1,
    is_expired=0,
    l2_origin=0
WHERE id=@user_pkg_id;

INSERT INTO cname_domains (domain, dns_provider_id, note, create_at, update_at)
SELECT 'cnn-ai.test', @dns_id, 'local bootstrap', @now, @now
WHERE NOT EXISTS (SELECT 1 FROM cname_domains WHERE domain='cnn-ai.test');

INSERT INTO config (name, value, type, scope_id, scope_name, create_at, update_at, enable)
SELECT 'edge_config_version', '1', 'system', 0, 'global', @now, @now, 1
WHERE NOT EXISTS (
    SELECT 1 FROM config
    WHERE name='edge_config_version' AND type='system' AND scope_id=0 AND scope_name='global'
);

INSERT INTO config (name, value, type, scope_id, scope_name, create_at, update_at, enable)
SELECT 'system_info',
       '{"sys_name":"CNN.NET","admin_console_title":"CNN.NET Admin","user_console_title":"CNN.NET User"}',
       'system', 0, 'global', @now, @now, 1
WHERE NOT EXISTS (
    SELECT 1 FROM config
    WHERE name='system_info' AND type='system' AND scope_id=0 AND scope_name='global'
);

CREATE TABLE IF NOT EXISTS site_conf_cache (
    site_id BIGINT NULL,
    templ_md5 VARCHAR(64) NULL,
    version INT NULL,
    data LONGTEXT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TEMPORARY TABLE IF NOT EXISTS site_conf_cache_latest AS
SELECT
    s.site_id,
    ANY_VALUE(s.templ_md5) AS templ_md5,
    ANY_VALUE(s.version) AS version,
    ANY_VALUE(s.data) AS data
FROM site_conf_cache s
INNER JOIN (
    SELECT site_id, MAX(COALESCE(version, 0)) AS max_version
    FROM site_conf_cache
    WHERE site_id IS NOT NULL
    GROUP BY site_id
) v ON v.site_id = s.site_id AND COALESCE(s.version, 0) = v.max_version
GROUP BY s.site_id;

INSERT INTO site_conf_cache_latest (site_id, templ_md5, version, data)
SELECT site_id, templ_md5, version, data
FROM site_conf_cache
WHERE site_id IS NULL;

DELETE FROM site_conf_cache;

INSERT INTO site_conf_cache (site_id, templ_md5, version, data)
SELECT site_id, templ_md5, version, data
FROM site_conf_cache_latest;

DROP TEMPORARY TABLE IF EXISTS site_conf_cache_latest;

SET @idx_exists := (
    SELECT COUNT(1)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'site_conf_cache'
      AND index_name = 'uk_site_conf_cache_site_id'
);
SET @ddl_sql := IF(
    @idx_exists = 0,
    'ALTER TABLE site_conf_cache ADD UNIQUE KEY uk_site_conf_cache_site_id (site_id)',
    'SELECT 1'
);
PREPARE stmt FROM @ddl_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
SQL

echo "MySQL bootstrap completed: ${MYSQL_USER}@${MYSQL_HOST}:${MYSQL_PORT}/${MYSQL_DB}"
echo "Admin: cnn_ai_admin / admin123"
echo "User:  cnn_ai_user / user123"
