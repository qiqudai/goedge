using Microsoft.Extensions.Logging;
using SqlSugar;
using Cnn.Domain.Entities;

namespace Cnn.Api.Data;

public static class RuntimeSchema
{
    public static void Ensure(ISqlSugarClient db, ILogger? logger = null)
    {
        EnsurePackageColumns(db, logger);
        EnsureUserPackageColumns(db, logger);
        EnsureTaskColumns(db, logger);
        EnsureOpLogTable(db, logger);
        EnsureOpLogIndexes(db, logger);
        EnsureAccessLogDownloadTable(db, logger);
        EnsureAccessLogDownloadIndexes(db, logger);
        EnsureCertColumns(db, logger);
        EnsureFinanceTables(db, logger);
        EnsureSiteConfCacheUniqueIndex(db, logger);
    }

    private static void EnsurePackageColumns(ISqlSugarClient db, ILogger? logger)
    {
        TryAddColumn(db, "package", "l2_origin", "boolean", true, logger);
    }

    private static void EnsureUserPackageColumns(ISqlSugarClient db, ILogger? logger)
    {
        TryAddColumn(db, "user_package", "l2_origin", "boolean", true, logger);
        TryAddColumn(db, "user_package", "version", "int(11)", true, logger, "1");
        TryAddColumn(db, "user_package", "is_expired", "boolean", true, logger);
    }

    private static void EnsureTaskColumns(ISqlSugarClient db, ILogger? logger)
    {
        TryAddColumn(db, "task", "targets_json", "longtext", true, logger);
    }

    private static void EnsureOpLogTable(ISqlSugarClient db, ILogger? logger)
    {
        if (db.DbMaintenance.IsAnyTable("op_log"))
        {
            return;
        }

        try
        {
            db.CodeFirst.InitTables<OpLog>();
            if (db.DbMaintenance.IsAnyTable("op_log"))
            {
                return;
            }
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to init table op_log");
        }

        try
        {
            if (db.CurrentConnectionConfig.DbType == DbType.Sqlite)
            {
                db.Ado.ExecuteCommand(
                    """
CREATE TABLE IF NOT EXISTS op_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid INTEGER NULL,
  type TEXT NULL,
  action TEXT NULL,
  content TEXT NULL,
  diff TEXT NULL,
  ip TEXT NULL,
  create_at DATETIME NULL,
  process TEXT NULL
)
"""
                );
            }
            else
            {
                db.Ado.ExecuteCommand(
                    """
CREATE TABLE IF NOT EXISTS op_log (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  uid BIGINT NULL,
  type VARCHAR(64) NULL,
  action VARCHAR(255) NULL,
  content TEXT NULL,
  diff LONGTEXT NULL,
  ip VARCHAR(255) NULL,
  create_at DATETIME NULL,
  process VARCHAR(255) NULL
)
"""
                );
            }
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to create fallback table op_log");
        }
    }

    private static void EnsureAccessLogDownloadTable(ISqlSugarClient db, ILogger? logger)
    {
        if (db.DbMaintenance.IsAnyTable("access_log_download"))
        {
            return;
        }

        try
        {
            db.CodeFirst.InitTables<AccessLogDownload>();
            if (db.DbMaintenance.IsAnyTable("access_log_download"))
            {
                return;
            }
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to init table access_log_download");
        }

        try
        {
            if (db.CurrentConnectionConfig.DbType == DbType.Sqlite)
            {
                db.Ado.ExecuteCommand(
                    """
CREATE TABLE IF NOT EXISTS access_log_download (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NULL,
  is_admin INTEGER NULL,
  scope TEXT NULL,
  state TEXT NULL,
  query_json TEXT NULL,
  file_name TEXT NULL,
  rows INTEGER NULL,
  error TEXT NULL,
  create_at DATETIME NULL,
  finish_at DATETIME NULL
)
"""
                );
            }
            else
            {
                db.Ado.ExecuteCommand(
                    """
CREATE TABLE IF NOT EXISTS access_log_download (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NULL,
  is_admin TINYINT(1) NULL,
  scope VARCHAR(32) NULL,
  state VARCHAR(32) NULL,
  query_json LONGTEXT NULL,
  file_name VARCHAR(255) NULL,
  rows BIGINT NULL,
  error TEXT NULL,
  create_at DATETIME NULL,
  finish_at DATETIME NULL
)
"""
                );
            }
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to create fallback table access_log_download");
        }
    }

    private static void EnsureOpLogIndexes(ISqlSugarClient db, ILogger? logger)
    {
        if (!db.DbMaintenance.IsAnyTable("op_log"))
        {
            return;
        }

        try
        {
            if (db.CurrentConnectionConfig.DbType == DbType.Sqlite)
            {
                db.Ado.ExecuteCommand("CREATE INDEX IF NOT EXISTS idx_op_log_create_at ON op_log(create_at)");
                db.Ado.ExecuteCommand("CREATE INDEX IF NOT EXISTS idx_op_log_uid_create_at ON op_log(uid, create_at)");
                return;
            }

            if (db.CurrentConnectionConfig.DbType == DbType.MySql)
            {
                EnsureMySqlIndex(db, "op_log", "idx_op_log_create_at", "ALTER TABLE op_log ADD INDEX idx_op_log_create_at (create_at)");
                EnsureMySqlIndex(db, "op_log", "idx_op_log_uid_create_at", "ALTER TABLE op_log ADD INDEX idx_op_log_uid_create_at (uid, create_at)");
            }
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to ensure indexes on op_log");
        }
    }

    private static void EnsureAccessLogDownloadIndexes(ISqlSugarClient db, ILogger? logger)
    {
        if (!db.DbMaintenance.IsAnyTable("access_log_download"))
        {
            return;
        }

        try
        {
            if (db.CurrentConnectionConfig.DbType == DbType.Sqlite)
            {
                db.Ado.ExecuteCommand("CREATE INDEX IF NOT EXISTS idx_access_log_download_user_create_at ON access_log_download(user_id, create_at)");
                db.Ado.ExecuteCommand("CREATE INDEX IF NOT EXISTS idx_access_log_download_state_create_at ON access_log_download(state, create_at)");
                return;
            }

            if (db.CurrentConnectionConfig.DbType == DbType.MySql)
            {
                EnsureMySqlIndex(db, "access_log_download", "idx_access_log_download_user_create_at", "ALTER TABLE access_log_download ADD INDEX idx_access_log_download_user_create_at (user_id, create_at)");
                EnsureMySqlIndex(db, "access_log_download", "idx_access_log_download_state_create_at", "ALTER TABLE access_log_download ADD INDEX idx_access_log_download_state_create_at (state, create_at)");
            }
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to ensure indexes on access_log_download");
        }
    }

    private static void EnsureCertColumns(ISqlSugarClient db, ILogger? logger)
    {
        TryAddColumn(db, "cert", "state", "varchar(255)", true, logger);
        TryAddColumn(db, "cert", "ret", "text", true, logger);
    }

    private static void EnsureFinanceTables(ISqlSugarClient db, ILogger? logger)
    {
        if (db.DbMaintenance.IsAnyTable("balance_ledger"))
        {
            return;
        }

        try
        {
            db.CodeFirst.InitTables<BalanceLedger>();
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to init finance table balance_ledger");
        }
    }

    private static void EnsureSiteConfCacheUniqueIndex(ISqlSugarClient db, ILogger? logger)
    {
        if (db.CurrentConnectionConfig.DbType != DbType.MySql)
        {
            return;
        }

        if (!db.DbMaintenance.IsAnyTable("site_conf_cache"))
        {
            return;
        }

        try
        {
            var indexExists = QueryScalarInt(
                db,
                "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'site_conf_cache' AND index_name = 'uk_site_conf_cache_site_id'");
            if (indexExists > 0)
            {
                return;
            }

            var dupCount = QueryScalarInt(
                db,
                "SELECT COUNT(*) FROM (SELECT site_id FROM site_conf_cache WHERE site_id IS NOT NULL GROUP BY site_id HAVING COUNT(*) > 1) t");
            if (dupCount > 0)
            {
                db.Ado.ExecuteCommand(
                    """
CREATE TABLE IF NOT EXISTS site_conf_cache_dedupe_tmp (
    site_id BIGINT NULL,
    templ_md5 VARCHAR(64) NULL,
    version INT NULL,
    data LONGTEXT NULL
)
"""
                );
                db.Ado.ExecuteCommand("TRUNCATE TABLE site_conf_cache_dedupe_tmp");
                db.Ado.ExecuteCommand(
                    """
INSERT INTO site_conf_cache_dedupe_tmp (site_id, templ_md5, version, data)
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
GROUP BY s.site_id
"""
                );
                db.Ado.ExecuteCommand(
                    "INSERT INTO site_conf_cache_dedupe_tmp (site_id, templ_md5, version, data) SELECT site_id, templ_md5, version, data FROM site_conf_cache WHERE site_id IS NULL");

                db.Ado.ExecuteCommand("DELETE FROM site_conf_cache");
                db.Ado.ExecuteCommand(
                    "INSERT INTO site_conf_cache (site_id, templ_md5, version, data) SELECT site_id, templ_md5, version, data FROM site_conf_cache_dedupe_tmp");
                db.Ado.ExecuteCommand("DROP TABLE IF EXISTS site_conf_cache_dedupe_tmp");
            }

            db.Ado.ExecuteCommand("ALTER TABLE site_conf_cache ADD UNIQUE KEY uk_site_conf_cache_site_id (site_id)");
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to ensure unique index uk_site_conf_cache_site_id on site_conf_cache");
        }
    }

    private static int QueryScalarInt(ISqlSugarClient db, string sql)
    {
        var table = db.Ado.GetDataTable(sql);
        if (table.Rows.Count == 0 || table.Columns.Count == 0)
        {
            return 0;
        }

        var value = table.Rows[0][0];
        if (value == null || value == DBNull.Value)
        {
            return 0;
        }

        return Convert.ToInt32(value);
    }

    private static void EnsureMySqlIndex(ISqlSugarClient db, string table, string indexName, string createSql)
    {
        var exists = QueryScalarInt(
            db,
            $"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = '{table}' AND index_name = '{indexName}'");
        if (exists > 0)
        {
            return;
        }

        db.Ado.ExecuteCommand(createSql);
    }

    private static void TryAddColumn(
        ISqlSugarClient db,
        string table,
        string column,
        string dataType,
        bool nullable,
        ILogger? logger,
        string? defaultValue = null)
    {
        if (db.DbMaintenance.IsAnyColumn(table, column))
        {
            return;
        }

        try
        {
            var info = new DbColumnInfo
            {
                DbColumnName = column,
                DataType = dataType,
                IsNullable = nullable
            };
            if (!string.IsNullOrWhiteSpace(defaultValue))
            {
                info.DefaultValue = defaultValue;
            }

            db.DbMaintenance.AddColumn(table, info);
        }
        catch (Exception ex)
        {
            logger?.LogWarning(ex, "Failed to add runtime column {Column} on {Table}", column, table);
        }
    }
}
