using System.Data.Common;
using System.Diagnostics;
using System.Globalization;
using Cnn.Api.Services.Stats;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using SqlSugar;
using AccessLogDownloadEntity = Cnn.Domain.Entities.AccessLogDownload;
using CertEntity = Cnn.Domain.Entities.Cert;
using IpSwitchLogEntity = Cnn.Domain.Entities.IpSwitchLog;
using LoginLogEntity = Cnn.Domain.Entities.LoginLog;
using NodeMonitorLogEntity = Cnn.Domain.Entities.NodeMonitorLog;
using OpLogEntity = Cnn.Domain.Entities.OpLog;
using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Common;

public sealed class CleanupAndBackupWorker : BackgroundService
{
    private static readonly TimeSpan Interval = TimeSpan.FromHours(1);
    private static readonly object BackupLock = new();
    private static DateTime _lastBackupAt = DateTime.MinValue;

    private readonly IServiceScopeFactory _scopeFactory;
    private readonly ILogger<CleanupAndBackupWorker> _logger;
    private readonly IConfiguration _configuration;

    public CleanupAndBackupWorker(IServiceScopeFactory scopeFactory, ILogger<CleanupAndBackupWorker> logger, IConfiguration configuration)
    {
        _scopeFactory = scopeFactory;
        _logger = logger;
        _configuration = configuration;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await RunOnceAsync(stoppingToken);
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "Cleanup/backup worker failed");
            }

            try
            {
                await Task.Delay(Interval, stoppingToken);
            }
            catch (TaskCanceledException)
            {
                return;
            }
        }
    }

    private async Task RunOnceAsync(CancellationToken cancellationToken)
    {
        using var scope = _scopeFactory.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<ISqlSugarClient>();
        var systemConfig = scope.ServiceProvider.GetRequiredService<ISystemConfigService>();

        var cfg = await systemConfig.LoadSystemConfigAsync(cancellationToken);
        await RunCleanupAsync(db, cfg, cancellationToken);
        await RunBackupAsync(db, cfg, cancellationToken);
    }

    private async Task RunCleanupAsync(ISqlSugarClient db, Dictionary<string, string> cfg, CancellationToken cancellationToken)
    {
        await CleanupTableByDaysAsync<LoginLogEntity>(db, "login_log", ResolveDays(cfg, 30, "keep-login-log-days"), cancellationToken);
        await CleanupTableByDaysAsync<OpLogEntity>(db, "op_log", ResolveDays(cfg, 30, "keep-op-log-days"), cancellationToken);
        await CleanupTasksByDaysAsync(db, ResolveDays(cfg, 7, "keep-task-log-days"), cancellationToken);
        await CleanupTableByDaysAsync<NodeMonitorLogEntity>(db, "node_monitor_log", ResolveDays(cfg, 7, "keep-node-log-days"), cancellationToken);
        await CleanupTableByDaysAsync<IpSwitchLogEntity>(
            db,
            "ip_switch_log",
            ResolveDays(cfg, 7, "keep-blacklist-days", "keep-node-log-days"),
            cancellationToken);
        await CleanupAccessLogDownloadsByDaysAsync(
            db,
            ResolveDays(cfg, 30, "keep-access-download-log-days", "keep-task-log-days"),
            cancellationToken);

        await CleanupIssueCertTasksAsync(db, cfg, cancellationToken);

        var accessDays = MinPositive(
            ResolveDays(cfg, 0, "keep-access-log-days"),
            ResolveDays(cfg, 0, "keep-traffic-history-days")
        );
        var clickHouse = ClickHouseHttpHelper.ResolveConfig(_configuration);
        await CleanupClickHouseByDaysAsync(clickHouse, "node_access_logs", accessDays, cancellationToken);
        await CleanupClickHouseByDaysAsync(clickHouse, "node_events", ResolveDays(cfg, 0, "keep-node-log-days"), cancellationToken);
        await CleanupClickHouseByDaysAsync(clickHouse, "node_metrics", ResolveDays(cfg, 0, "keep-node-traffic-days", "clean_node_traffic_days"), cancellationToken);
    }

    private async Task CleanupIssueCertTasksAsync(ISqlSugarClient db, Dictionary<string, string> cfg, CancellationToken cancellationToken)
    {
        var timeoutMinutes = ResolveInt(cfg, 120, "cert_issue_timeout_minutes");
        if (timeoutMinutes <= 0)
        {
            return;
        }

        var cutoff = DateTime.Now.AddMinutes(-timeoutMinutes);
        var tasks = await db.Queryable<TaskEntity>()
            .Where(t => t.Type == "issue_cert" && t.Enable == true)
            .Where(t => t.State == "waiting" || t.State == "running" || t.State == "retrying")
            .Where(t =>
                (t.StartAt == null && t.CreateAt != null && t.CreateAt < cutoff) ||
                (t.StartAt != null && t.StartAt < cutoff))
            .ToListAsync();

        if (tasks.Count == 0)
        {
            return;
        }

        var now = DateTime.Now;
        var reason = $"证书签发超时（超过 {timeoutMinutes} 分钟）";
        var ids = tasks.Select(t => t.Id).Where(id => id > 0).ToList();
        if (ids.Count == 0)
        {
            return;
        }

        await db.Updateable<TaskEntity>()
            .SetColumns(t => new TaskEntity
            {
                State = "fail",
                Enable = false,
                Ret = reason,
                EndAt = now,
                RetryAt = null
            })
            .Where(t => ids.Contains(t.Id))
            .ExecuteCommandAsync();

        await db.Updateable<CertEntity>()
            .SetColumns(c => new CertEntity
            {
                State = "fail",
                Ret = reason,
                UpdateAt = now
            })
            .Where(c => c.IssueTaskId != null && ids.Contains(c.IssueTaskId.Value))
            .ExecuteCommandAsync();

        _logger.LogWarning("Issue cert tasks timed out: {Count}", ids.Count);
    }

    private static async Task CleanupTableByDaysAsync<T>(ISqlSugarClient db, string tableName, int days, CancellationToken cancellationToken) where T : class, new()
    {
        if (days <= 0)
        {
            return;
        }

        if (!string.IsNullOrWhiteSpace(tableName) && !db.DbMaintenance.IsAnyTable(tableName))
        {
            return;
        }

        var cutoff = DateTime.Now.AddDays(-days);
        await db.Deleteable<T>()
            .Where("create_at < @cutoff", new { cutoff })
            .ExecuteCommandAsync();
    }

    private static async Task CleanupTasksByDaysAsync(ISqlSugarClient db, int days, CancellationToken cancellationToken)
    {
        if (days <= 0)
        {
            return;
        }

        if (!db.DbMaintenance.IsAnyTable("task"))
        {
            return;
        }

        var cutoff = DateTime.Now.AddDays(-days);
        const string sql = """
                           DELETE FROM task
                           WHERE create_at < @cutoff
                             AND NOT EXISTS (SELECT 1 FROM cert WHERE cert.task_id = task.id OR cert.issue_task_id = task.id)
                             AND NOT EXISTS (SELECT 1 FROM config WHERE config.task_id = task.id)
                             AND NOT EXISTS (SELECT 1 FROM line WHERE line.task_id = task.id)
                             AND NOT EXISTS (SELECT 1 FROM site WHERE site.task_id = task.id OR site.cname_task_id = task.id)
                             AND NOT EXISTS (SELECT 1 FROM stream WHERE stream.task_id = task.id OR stream.cname_task_id = task.id)
                             AND NOT EXISTS (SELECT 1 FROM user_package WHERE user_package.task_id = task.id)
                           """;

        await db.Ado.ExecuteCommandAsync(sql, new { cutoff });
    }

    private static async Task CleanupAccessLogDownloadsByDaysAsync(ISqlSugarClient db, int days, CancellationToken cancellationToken)
    {
        if (days <= 0)
        {
            return;
        }

        if (!db.DbMaintenance.IsAnyTable("access_log_download"))
        {
            return;
        }

        var cutoff = DateTime.Now.AddDays(-days);
        await db.Deleteable<AccessLogDownloadEntity>()
            .Where("COALESCE(finish_at, create_at) < @cutoff", new { cutoff })
            .ExecuteCommandAsync();
    }

    private static async Task CleanupClickHouseByDaysAsync(
        ClickHouseHttpConfig? clickHouse,
        string table,
        int days,
        CancellationToken cancellationToken)
    {
        if (days <= 0)
        {
            return;
        }

        if (clickHouse == null)
        {
            return;
        }

        var cutoff = DateTime.Now.AddDays(-days).ToString("yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
        var query = $"ALTER TABLE {table} DELETE WHERE ts < toDateTime('{cutoff}')";
        await ClickHouseHttpHelper.ExecuteAsync(clickHouse, query, cancellationToken);
    }

    private async Task RunBackupAsync(ISqlSugarClient db, Dictionary<string, string> cfg, CancellationToken cancellationToken)
    {
        var backupDir = ResolveString(cfg, "backup_dir");
        if (string.IsNullOrWhiteSpace(backupDir))
        {
            return;
        }

        var interval = ResolveBackupInterval(cfg);
        if (interval <= TimeSpan.Zero)
        {
            return;
        }

        var now = DateTime.Now;
        lock (BackupLock)
        {
            if (_lastBackupAt != DateTime.MinValue && now - _lastBackupAt < interval)
            {
                return;
            }
        }

        var startAt = DateTime.Now;
        var path = string.Empty;
        Exception? error = null;
        try
        {
            backupDir = ResolveBasePath(backupDir);
            path = await RunDatabaseBackupAsync(backupDir, cancellationToken);
        }
        catch (Exception ex)
        {
            error = ex;
            _logger.LogWarning(ex, "Database backup failed");
        }

        var finishAt = DateTime.Now;
        var success = error == null;
        var result = success ? path : (error?.Message ?? "backup failed");
        await RecordBackupTaskAsync(db, startAt, finishAt, success, result, cancellationToken);

        lock (BackupLock)
        {
            _lastBackupAt = finishAt;
        }

        var keepDays = ResolveDays(cfg, 7, "backup_keep_days", "backup_retention");
        CleanupBackupFiles(backupDir, keepDays);
    }

    private async Task<string> RunDatabaseBackupAsync(string backupDir, CancellationToken cancellationToken)
    {
        Directory.CreateDirectory(backupDir);

        var connString = _configuration.GetConnectionString("Default")
            ?? _configuration["ConnectionStrings:Default"]
            ?? string.Empty;
        if (!TryParseConnectionString(connString, out var info))
        {
            throw new InvalidOperationException("invalid database connection string");
        }

        var filename = $"backup-{DateTime.Now:yyyyMMdd-HHmmss}.sql";
        var path = Path.Combine(backupDir, filename);
        await using var file = new FileStream(path, FileMode.Create, FileAccess.Write, FileShare.None);

        var psi = new ProcessStartInfo("mysqldump")
        {
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false
        };
        psi.ArgumentList.Add("-h");
        psi.ArgumentList.Add(info.Host);
        psi.ArgumentList.Add("-P");
        psi.ArgumentList.Add(info.Port);
        psi.ArgumentList.Add("-u");
        psi.ArgumentList.Add(info.User);
        psi.ArgumentList.Add(info.Database);

        if (!string.IsNullOrWhiteSpace(info.Password))
        {
            psi.Environment["MYSQL_PWD"] = info.Password;
        }

        using var process = Process.Start(psi);
        if (process == null)
        {
            throw new InvalidOperationException("mysqldump start failed");
        }

        var copyTask = process.StandardOutput.BaseStream.CopyToAsync(file, cancellationToken);
        var errorTask = process.StandardError.ReadToEndAsync(cancellationToken);
        await Task.WhenAll(copyTask, process.WaitForExitAsync(cancellationToken));
        var stderr = await errorTask;
        if (process.ExitCode != 0)
        {
            TryDeleteFile(path);
            throw new InvalidOperationException(string.IsNullOrWhiteSpace(stderr) ? "mysqldump failed" : stderr.Trim());
        }

        return path;
    }

    private static async Task RecordBackupTaskAsync(
        ISqlSugarClient db,
        DateTime startAt,
        DateTime finishAt,
        bool success,
        string result,
        CancellationToken cancellationToken)
    {
        var task = new TaskEntity
        {
            Name = "database_backup",
            Type = "backup",
            CreateAt = startAt,
            StartAt = startAt,
            EndAt = finishAt,
            Ret = result,
            State = success ? "done" : "fail",
            Enable = true,
            ErrTimes = success ? 0 : 1
        };
        await db.Insertable(task).ExecuteCommandAsync();
    }

    private static void CleanupBackupFiles(string dir, int keepDays)
    {
        if (keepDays <= 0)
        {
            return;
        }

        if (!Directory.Exists(dir))
        {
            return;
        }

        var cutoff = DateTime.Now.AddDays(-keepDays);
        foreach (var file in Directory.EnumerateFiles(dir))
        {
            try
            {
                var info = new FileInfo(file);
                if (info.LastWriteTime < cutoff)
                {
                    info.Delete();
                }
            }
            catch
            {
                // ignore cleanup failure
            }
        }
    }

    private static void TryDeleteFile(string path)
    {
        try
        {
            if (File.Exists(path))
            {
                File.Delete(path);
            }
        }
        catch
        {
            // ignore
        }
    }

    private static TimeSpan ResolveBackupInterval(Dictionary<string, string> cfg)
    {
        var raw = ResolveString(cfg, "backup_rate", "backup_frequency");
        return ParseBackupInterval(raw);
    }

    private static TimeSpan ParseBackupInterval(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return TimeSpan.Zero;
        }

        raw = raw.Trim();
        if (raw.EndsWith("d", StringComparison.OrdinalIgnoreCase))
        {
            var token = raw[..^1].Trim();
            if (int.TryParse(token, out var days) && days > 0)
            {
                return TimeSpan.FromDays(days);
            }
        }

        if (TimeSpan.TryParse(raw, out var duration))
        {
            return duration;
        }

        if (int.TryParse(raw, out var fallbackDays) && fallbackDays > 0)
        {
            return TimeSpan.FromDays(fallbackDays);
        }

        return TimeSpan.Zero;
    }

    private static int ResolveDays(Dictionary<string, string> cfg, int fallback, params string[] keys)
    {
        var raw = ResolveString(cfg, keys);
        if (string.IsNullOrWhiteSpace(raw))
        {
            return fallback;
        }

        if (!int.TryParse(raw.Trim(), out var value) || value < 0)
        {
            return fallback;
        }

        return value;
    }

    private static int ResolveInt(Dictionary<string, string> cfg, int fallback, params string[] keys)
    {
        var raw = ResolveString(cfg, keys);
        if (string.IsNullOrWhiteSpace(raw))
        {
            return fallback;
        }

        if (!int.TryParse(raw.Trim(), out var value))
        {
            return fallback;
        }

        return value;
    }

    private static string ResolveString(Dictionary<string, string> cfg, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (string.IsNullOrWhiteSpace(key))
            {
                continue;
            }

            if (cfg.TryGetValue(key, out var value) && !string.IsNullOrWhiteSpace(value))
            {
                return value;
            }
        }

        return string.Empty;
    }

    private static int MinPositive(params int[] values)
    {
        var min = 0;
        foreach (var value in values)
        {
            if (value <= 0)
            {
                continue;
            }
            if (min == 0 || value < min)
            {
                min = value;
            }
        }

        return min;
    }

    private static string ResolveBasePath(string path)
    {
        var trimmed = path.Trim();
        if (Path.IsPathRooted(trimmed))
        {
            return trimmed;
        }

        var baseDir = AppContext.BaseDirectory;
        if (string.IsNullOrWhiteSpace(baseDir))
        {
            var processPath = Environment.ProcessPath;
            if (!string.IsNullOrWhiteSpace(processPath))
            {
                baseDir = Path.GetDirectoryName(processPath) ?? string.Empty;
            }
        }

        return Path.Combine(baseDir, trimmed);
    }

    private sealed record MySqlInfo(string Host, string Port, string User, string Password, string Database);

    private static bool TryParseConnectionString(string raw, out MySqlInfo info)
    {
        info = null!;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        var builder = new DbConnectionStringBuilder { ConnectionString = raw };
        var host = GetValue(builder, "server", "host", "data source", "datasource", "addr", "address");
        var port = GetValue(builder, "port", "server port");
        var user = GetValue(builder, "user id", "uid", "user", "username");
        var password = GetValue(builder, "password", "pwd");
        var database = GetValue(builder, "database", "initial catalog", "dbname");

        if (string.IsNullOrWhiteSpace(host))
        {
            host = "127.0.0.1";
        }

        if (string.IsNullOrWhiteSpace(port))
        {
            port = "3306";
        }

        if (string.IsNullOrWhiteSpace(database))
        {
            return false;
        }

        info = new MySqlInfo(host, port, user, password, database);
        return true;
    }

    private static string GetValue(DbConnectionStringBuilder builder, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (builder.TryGetValue(key, out var value))
            {
                return Convert.ToString(value, CultureInfo.InvariantCulture) ?? string.Empty;
            }
        }

        return string.Empty;
    }
}
