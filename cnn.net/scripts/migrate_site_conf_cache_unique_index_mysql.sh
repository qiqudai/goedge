#!/usr/bin/env bash
set -euo pipefail

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-123456}"
MYSQL_DB="${MYSQL_DB:-cdnfy}"

MYSQL_BASE_ARGS=(-h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" "-p${MYSQL_PASSWORD}" --default-character-set=utf8mb4)

mysql "${MYSQL_BASE_ARGS[@]}" "${MYSQL_DB}" <<'SQL'
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

echo "Migration completed: uk_site_conf_cache_site_id ensured on ${MYSQL_DB}.site_conf_cache"
