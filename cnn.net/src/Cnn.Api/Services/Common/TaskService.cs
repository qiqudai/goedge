using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;
using Cnn.Api.Services.Common.Tasks;

namespace Cnn.Api.Services.Common;

public sealed class TaskService : ITaskService
{
    private const int DefaultPageSize = 10;
    private const int MaxPageSize = 500;
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly ITaskMetadataAccessor _metadataAccessor;

    public TaskService(ISqlSugarClient db, ITaskMetadataAccessor metadataAccessor)
    {
        _db = db;
        _metadataAccessor = metadataAccessor;
    }

    public Task<ServiceResult<TaskListResult>> ListAdminAsync(TaskListQuery query, CancellationToken cancellationToken)
    {
        return ListInternalAsync(query, null, cancellationToken);
    }

    public Task<ServiceResult<TaskListResult>> ListUserAsync(TaskListQuery query, long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return Task.FromResult(ServiceResult<TaskListResult>.Fail(ErrorCodes.InvalidParam, "user_id_required"));
        }

        return ListInternalAsync(query, userId, cancellationToken);
    }

    public async Task<ServiceResult<TaskDetailDto>> GetAdminAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<TaskDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        var task = await _db.Queryable<TaskEntity>().Where(t => t.Id == id).FirstAsync();
        if (task == null)
        {
            return ServiceResult<TaskDetailDto>.Fail(ErrorCodes.NotFound, "task_not_found");
        }

        return ServiceResult<TaskDetailDto>.Ok(BuildDetail(task));
    }

    public async Task<ServiceResult<TaskDetailDto>> GetUserAsync(long id, long userId, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<TaskDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        if (userId <= 0)
        {
            return ServiceResult<TaskDetailDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var task = await _db.Queryable<TaskEntity>().Where(t => t.Id == id).FirstAsync();
        if (task == null)
        {
            return ServiceResult<TaskDetailDto>.Fail(ErrorCodes.NotFound, "task_not_found");
        }

        if (!await CanAccessTaskAsync(task, userId))
        {
            return ServiceResult<TaskDetailDto>.Fail(ErrorCodes.PermissionDenied);
        }

        return ServiceResult<TaskDetailDto>.Ok(BuildDetail(task));
    }

    public async Task<ServiceResult<TaskUsagePayload>> GetUsageAsync(long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<TaskUsagePayload>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var limits = await LoadPurgeLimitsAsync(cancellationToken);
        if (!limits.Ok)
        {
            return ServiceResult<TaskUsagePayload>.Fail(ErrorCodes.DbError, "db_error");
        }

        var usage = await LoadUserPurgeUsageAsync(userId, cancellationToken);
        if (!usage.Ok || usage.Value == null)
        {
            return ServiceResult<TaskUsagePayload>.Fail(ErrorCodes.DbError, "db_error");
        }

        var remaining = new TaskUsageLimit
        {
            RefreshUrl = CalcRemaining(limits.Value.RefreshUrl, usage.Value.RefreshUrl),
            RefreshDir = CalcRemaining(limits.Value.RefreshDir, usage.Value.RefreshDir),
            Preheat = CalcRemaining(limits.Value.Preheat, usage.Value.Preheat)
        };

        return ServiceResult<TaskUsagePayload>.Ok(new TaskUsagePayload(limits.Value, usage.Value, remaining));
    }

    public async Task<ServiceResult<bool>> CreateAsync(
        TaskCreateRequest request,
        long userId,
        bool adminMode,
        CancellationToken cancellationToken)
    {
        var taskType = request?.Type?.Trim().ToLowerInvariant();
        if (!IsPurgeType(taskType))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "task.type_invalid");
        }

        var urls = SplitTaskLines(request?.Urls);
        if (urls.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "task.urls_required");
        }

        var resolvedUserId = ResolveUserId(request?.UserId, userId, adminMode);
        if (resolvedUserId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var normalizeResult = await NormalizePurgeUrlsAsync(urls, adminMode, resolvedUserId, cancellationToken);
        if (!normalizeResult.Ok)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, normalizeResult.ErrorKey);
        }

        var quotaResult = await ConsumePurgeQuotaAsync(resolvedUserId, taskType!, normalizeResult.Value!.Count, cancellationToken);
        if (!quotaResult.Ok)
        {
            return ServiceResult<bool>.Fail(quotaResult.ErrorCode, quotaResult.ErrorKey);
        }

        var metaRaw = _metadataAccessor.BuildOwnerMeta(resolvedUserId);

        var task = new TaskEntity
        {
            Type = taskType,
            Data = string.Join('\n', normalizeResult.Value),
            Res = metaRaw,
            State = "waiting",
            CreateAt = DateTime.Now,
            Enable = true
        };

        var inserted = await _db.Insertable(task).ExecuteCommandAsync();
        if (inserted <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> ResubmitAsync(
        long id,
        long userId,
        bool adminMode,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var task = await _db.Queryable<TaskEntity>().Where(t => t.Id == id).FirstAsync();
        if (task == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "task_not_found");
        }

        var ownerId = _metadataAccessor.GetOwnerUserId(task);
        if (!adminMode || userId <= 0)
        {
            ownerId = userId;
        }

        if (ownerId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var urls = SplitTaskLines(task.Data);
        if (urls.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "task.urls_required");
        }

        var normalizeResult = await NormalizePurgeUrlsAsync(urls, adminMode, ownerId, cancellationToken);
        if (!normalizeResult.Ok)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, normalizeResult.ErrorKey);
        }

        var quotaResult = await ConsumePurgeQuotaAsync(ownerId, task.Type ?? string.Empty, normalizeResult.Value!.Count, cancellationToken);
        if (!quotaResult.Ok)
        {
            return ServiceResult<bool>.Fail(quotaResult.ErrorCode, quotaResult.ErrorKey);
        }

        var metaRaw = _metadataAccessor.BuildOwnerMeta(ownerId);

        var newTask = new TaskEntity
        {
            Type = task.Type,
            Data = string.Join('\n', normalizeResult.Value),
            Res = metaRaw,
            State = "waiting",
            CreateAt = DateTime.Now,
            Enable = true
        };

        var inserted = await _db.Insertable(newTask).ExecuteCommandAsync();
        if (inserted <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    private async Task<ServiceResult<TaskListResult>> ListInternalAsync(
        TaskListQuery query,
        long? fixedUserId,
        CancellationToken cancellationToken)
    {
        query ??= new TaskListQuery();
        var (page, pageSize) = ResolvePaging(query);
        var keyword = query.Keyword?.Trim();
        var taskType = query.Type?.Trim();
        var userId = fixedUserId ?? query.UserId;

        var q = _db.Queryable<TaskEntity>();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            q = q.Where(t => SqlFunc.Contains(t.Data, keyword));
        }

        if (!string.IsNullOrWhiteSpace(taskType))
        {
            q = q.Where(t => t.Type == taskType);
        }

        if (userId.HasValue && userId.Value > 0)
        {
            var pattern = $"\"user_id\":{userId.Value}";
            q = q.Where(t => SqlFunc.Contains(t.Res, pattern));
        }

        var total = await q.CountAsync();
        var list = await q.OrderBy(t => t.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .Select(t => new TaskListItemDto
            {
                Id = t.Id,
                Pid = t.Pid,
                Pry = t.Pry,
                Name = t.Name,
                Data = t.Data,
                Type = t.Type,
                Depend = t.Depend,
                CreateAt = t.CreateAt,
                StartAt = t.StartAt,
                EndAt = t.EndAt,
                State = t.State,
                ErrTimes = t.ErrTimes,
                RetryAt = t.RetryAt,
                Ret = t.Ret,
                TargetsJson = t.TargetsJson,
                Progress = t.Progress
            })
            .ToListAsync();

        return ServiceResult<TaskListResult>.Ok(new TaskListResult(list, total, page));
    }

    private static TaskDetailDto BuildDetail(TaskEntity task)
    {
        return new TaskDetailDto
        {
            Id = task.Id,
            Pid = task.Pid,
            Pry = task.Pry,
            Name = task.Name,
            Type = task.Type,
            Depend = task.Depend,
            CreateAt = task.CreateAt,
            StartAt = task.StartAt,
            EndAt = task.EndAt,
            State = task.State,
            ErrTimes = task.ErrTimes,
            Progress = task.Progress,
            Ret = task.Ret
        };
    }

    private async Task<bool> CanAccessTaskAsync(TaskEntity task, long userId)
    {
        var ownerId = _metadataAccessor.GetOwnerUserId(task);
        if (ownerId > 0)
        {
            return ownerId == userId;
        }

        if (!string.Equals(task.Type, "clear_cache", StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        var siteIds = _metadataAccessor.GetSiteIds(task);
        if (siteIds.Count == 0)
        {
            return false;
        }

        var allowed = await _db.Queryable<Site>()
            .Where(s => s.Uid == (int)userId && siteIds.Contains(s.Id))
            .Select(s => s.Id)
            .ToListAsync();

        return allowed.Count == siteIds.Count;
    }

    private async Task<(bool Ok, TaskUsageLimit Value)> LoadPurgeLimitsAsync(CancellationToken cancellationToken)
    {
        var limits = new TaskUsageLimit
        {
            RefreshUrl = 2000,
            RefreshDir = 500,
            Preheat = 2000
        };

        var configs = await _db.Queryable<Config>()
            .Where(c => c.Type == "site" && c.Name != null && new[] { "clean_url", "clean_dir", "pre_cache_url" }.Contains(c.Name))
            .ToListAsync();

        foreach (var cfg in configs)
        {
            if (!int.TryParse(cfg.Value?.Trim(), out var value) || value <= 0)
            {
                continue;
            }

            switch (cfg.Name)
            {
                case "clean_url":
                    limits.RefreshUrl = value;
                    break;
                case "clean_dir":
                    limits.RefreshDir = value;
                    break;
                case "pre_cache_url":
                    limits.Preheat = value;
                    break;
            }
        }

        return (true, limits);
    }

    private async Task<(bool Ok, TaskUsage? Value)> LoadUserPurgeUsageAsync(long userId, CancellationToken cancellationToken)
    {
        var today = DateTime.Now.ToString("yyyy-MM-dd");
        var usage = new TaskUsage { Date = today };

        var cfg = await _db.Queryable<Config>()
            .Where(c => c.Name == "purge_usage" && c.Type == "user" && c.ScopeName == "user" && c.ScopeId == userId)
            .FirstAsync();

        if (cfg == null || string.IsNullOrWhiteSpace(cfg.Value))
        {
            return (true, usage);
        }

        try
        {
            var parsed = JsonSerializer.Deserialize<TaskUsage>(cfg.Value, JsonOptions);
            if (parsed == null || string.IsNullOrWhiteSpace(parsed.Date))
            {
                return (true, usage);
            }

            if (!string.Equals(parsed.Date, today, StringComparison.Ordinal))
            {
                return (true, usage);
            }

            return (true, parsed);
        }
        catch
        {
            return (true, usage);
        }
    }

    private async Task<(bool Ok, int ErrorCode, string? ErrorKey)> ConsumePurgeQuotaAsync(
        long userId,
        string taskType,
        int count,
        CancellationToken cancellationToken)
    {
        if (count <= 0)
        {
            return (true, ErrorCodes.Success, null);
        }

        var limits = await LoadPurgeLimitsAsync(cancellationToken);
        if (!limits.Ok)
        {
            return (false, ErrorCodes.DbError, "db_error");
        }

        var usage = await LoadUserPurgeUsageAsync(userId, cancellationToken);
        if (!usage.Ok || usage.Value == null)
        {
            return (false, ErrorCodes.DbError, "db_error");
        }

        var current = usage.Value;
        switch (taskType)
        {
            case "refresh_url":
                if (ExceedsLimit(limits.Value.RefreshUrl, current.RefreshUrl, count))
                {
                    return (false, ErrorCodes.QuotaExceeded, "task.today_refresh_url_limit");
                }
                current.RefreshUrl += count;
                break;
            case "refresh_dir":
                if (ExceedsLimit(limits.Value.RefreshDir, current.RefreshDir, count))
                {
                    return (false, ErrorCodes.QuotaExceeded, "task.today_refresh_dir_limit");
                }
                current.RefreshDir += count;
                break;
            case "preheat":
                if (ExceedsLimit(limits.Value.Preheat, current.Preheat, count))
                {
                    return (false, ErrorCodes.QuotaExceeded, "task.today_preheat_limit");
                }
                current.Preheat += count;
                break;
        }

        var saved = await SaveUserPurgeUsageAsync(userId, current, cancellationToken);
        if (!saved)
        {
            return (false, ErrorCodes.DbError, "db_save_error");
        }

        return (true, ErrorCodes.Success, null);
    }

    private async Task<bool> SaveUserPurgeUsageAsync(long userId, TaskUsage usage, CancellationToken cancellationToken)
    {
        usage.Date = DateTime.Now.ToString("yyyy-MM-dd");
        var raw = JsonSerializer.Serialize(usage, JsonOptions);

        var query = _db.Queryable<Config>()
            .Where(c => c.Name == "purge_usage" && c.Type == "user" && c.ScopeName == "user" && c.ScopeId == userId);

        var cfg = await query.FirstAsync();
        if (cfg == null)
        {
            cfg = new Config
            {
                Name = "purge_usage",
                Value = raw,
                Type = "user",
                ScopeName = "user",
                ScopeId = (int)userId,
                Enable = true,
                CreateAt = DateTime.Now,
                UpdateAt = DateTime.Now
            };

            return await _db.Insertable(cfg).ExecuteCommandAsync() > 0;
        }

        cfg.Value = raw;
        cfg.UpdateAt = DateTime.Now;
        return await _db.Updateable(cfg)
            .UpdateColumns(c => new { c.Value, c.UpdateAt })
            .Where(c => c.Name == "purge_usage" && c.Type == "user" && c.ScopeName == "user" && c.ScopeId == userId)
            .ExecuteCommandAsync() > 0;
    }

    private static (int Page, int PageSize) ResolvePaging(TaskListQuery query)
    {
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        return (page, pageSize);
    }

    private static bool IsPurgeType(string? value)
    {
        return value is "refresh_url" or "refresh_dir" or "preheat";
    }

    private static List<string> SplitTaskLines(string? input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return new List<string>();
        }

        var normalized = input.Replace("\r\n", "\n", StringComparison.Ordinal)
            .Replace("\r", "\n", StringComparison.Ordinal);
        var parts = normalized.Split('\n', StringSplitOptions.RemoveEmptyEntries);
        var list = new List<string>();
        foreach (var part in parts)
        {
            var trimmed = part.Trim();
            if (!string.IsNullOrWhiteSpace(trimmed))
            {
                list.Add(trimmed);
            }
        }

        return list;
    }

    private async Task<(bool Ok, List<string>? Value, string? ErrorKey)> NormalizePurgeUrlsAsync(
        IReadOnlyList<string> urls,
        bool adminMode,
        long userId,
        CancellationToken cancellationToken)
    {
        var known = await LoadSiteDomainsAsync(adminMode, userId, cancellationToken);
        if (known == null)
        {
            return (false, null, "invalid_param");
        }

        var output = new List<string>(urls.Count);
        foreach (var raw in urls)
        {
            if (!raw.StartsWith("http://", StringComparison.OrdinalIgnoreCase) &&
                !raw.StartsWith("https://", StringComparison.OrdinalIgnoreCase))
            {
                return (false, null, "task.url_invalid_prefix");
            }

            if (!TryParseUrl(raw, out var scheme, out var host, out var port, out var rest))
            {
                return (false, null, "task.url_invalid_format");
            }

            if (host.Contains('*'))
            {
                if (!host.StartsWith("*.", StringComparison.Ordinal))
                {
                    return (false, null, "task.wildcard_invalid");
                }

                var suffix = host[2..];
                if (string.IsNullOrWhiteSpace(suffix))
                {
                    return (false, null, "task.wildcard_invalid");
                }

                var matches = MatchWildcardDomains(known, suffix);
                if (matches.Count == 0)
                {
                    return (false, null, "task.domain_not_found");
                }

                foreach (var match in matches)
                {
                    var target = BuildUrl(scheme, match, port, rest);
                    output.Add(target);
                }

                continue;
            }

            if (!IsKnownDomain(known, host, port))
            {
                return (false, null, "task.domain_not_found");
            }

            output.Add(raw);
        }

        return (true, output, null);
    }

    private async Task<KnownDomainSet?> LoadSiteDomainsAsync(bool adminMode, long userId, CancellationToken cancellationToken)
    {
        if (!adminMode && userId <= 0)
        {
            return null;
        }

        var query = _db.Queryable<Site>();
        if (!adminMode && userId > 0)
        {
            query = query.Where(s => s.Uid == (int)userId);
        }

        var sites = await query.Select(s => new { s.Domain }).ToListAsync();
        var set = new KnownDomainSet();
        foreach (var site in sites)
        {
            foreach (var domain in DomainParser.ParseDomains(site.Domain))
            {
                var trimmed = domain.Trim();
                if (string.IsNullOrWhiteSpace(trimmed))
                {
                    continue;
                }

                set.Exact.Add(trimmed);
                var host = SplitDomainHost(trimmed);
                if (!string.IsNullOrWhiteSpace(host))
                {
                    set.Host.Add(host);
                }
            }
        }

        return set;
    }

    private static string SplitDomainHost(string raw)
    {
        var host = raw.Trim();
        if (host.Contains("://", StringComparison.Ordinal))
        {
            if (Uri.TryCreate(host, UriKind.Absolute, out var uri) && !string.IsNullOrWhiteSpace(uri.Host))
            {
                return uri.Host;
            }
        }

        var stopIndex = host.IndexOfAny(new[] { '/', '?', '#' });
        if (stopIndex >= 0)
        {
            host = host[..stopIndex];
        }

        var colonIndex = host.LastIndexOf(':');
        if (colonIndex > 0 && colonIndex < host.Length - 1)
        {
            var port = host[(colonIndex + 1)..];
            if (int.TryParse(port, out _))
            {
                host = host[..colonIndex];
            }
        }

        return host.Trim();
    }

    private static bool TryParseUrl(string raw, out string scheme, out string host, out string port, out string rest)
    {
        scheme = string.Empty;
        host = string.Empty;
        port = string.Empty;
        rest = string.Empty;

        var schemeIndex = raw.IndexOf("://", StringComparison.Ordinal);
        if (schemeIndex <= 0)
        {
            return false;
        }

        scheme = raw[..schemeIndex].ToLowerInvariant();
        var remain = raw[(schemeIndex + 3)..];
        if (string.IsNullOrWhiteSpace(remain))
        {
            return false;
        }

        var splitIndex = remain.IndexOfAny(new[] { '/', '?', '#' });
        var hostPort = splitIndex >= 0 ? remain[..splitIndex] : remain;
        rest = splitIndex >= 0 ? remain[splitIndex..] : string.Empty;

        if (string.IsNullOrWhiteSpace(hostPort))
        {
            return false;
        }

        if (hostPort.StartsWith("[", StringComparison.Ordinal))
        {
            var end = hostPort.IndexOf(']');
            if (end < 0)
            {
                return false;
            }

            host = hostPort[..(end + 1)];
            if (end + 1 < hostPort.Length && hostPort[end + 1] == ':')
            {
                port = hostPort[(end + 2)..];
            }

            return true;
        }

        var colonIndex = hostPort.LastIndexOf(':');
        if (colonIndex > 0 && colonIndex < hostPort.Length - 1)
        {
            var portValue = hostPort[(colonIndex + 1)..];
            if (int.TryParse(portValue, out _))
            {
                host = hostPort[..colonIndex];
                port = portValue;
                return true;
            }
        }

        host = hostPort;
        return true;
    }

    private static bool IsKnownDomain(KnownDomainSet known, string host, string port)
    {
        if (!string.IsNullOrWhiteSpace(port))
        {
            var withPort = host + ":" + port;
            if (known.Exact.Contains(withPort))
            {
                return true;
            }
        }

        if (known.Exact.Contains(host))
        {
            return true;
        }

        return known.Host.Contains(host);
    }

    private static List<string> MatchWildcardDomains(KnownDomainSet known, string suffix)
    {
        var matches = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var dotted = "." + suffix;
        foreach (var domain in known.Exact)
        {
            var host = SplitDomainHost(domain);
            if (string.IsNullOrWhiteSpace(host))
            {
                continue;
            }

            if (string.Equals(host, suffix, StringComparison.OrdinalIgnoreCase) ||
                host.EndsWith(dotted, StringComparison.OrdinalIgnoreCase))
            {
                matches.Add(host);
            }
        }

        return matches.OrderBy(item => item).ToList();
    }

    private static string BuildUrl(string scheme, string host, string port, string rest)
    {
        if (string.IsNullOrWhiteSpace(port))
        {
            return $"{scheme}://{host}{rest}";
        }

        return $"{scheme}://{host}:{port}{rest}";
    }

    private static bool ExceedsLimit(int limit, int used, int add)
    {
        if (limit <= 0)
        {
            return false;
        }

        return used + add > limit;
    }

    private static int CalcRemaining(int limit, int used)
    {
        if (limit <= 0)
        {
            return 0;
        }

        var remain = limit - used;
        return remain < 0 ? 0 : remain;
    }

    private static long ResolveUserId(long? requested, long current, bool adminMode)
    {
        if (adminMode && requested.HasValue && requested.Value > 0)
        {
            return requested.Value;
        }

        return current;
    }

    private sealed class KnownDomainSet
    {
        public HashSet<string> Exact { get; } = new(StringComparer.OrdinalIgnoreCase);
        public HashSet<string> Host { get; } = new(StringComparer.OrdinalIgnoreCase);
    }
}
