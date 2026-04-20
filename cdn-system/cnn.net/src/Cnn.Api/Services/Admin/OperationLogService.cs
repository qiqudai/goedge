using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public interface IOperationLogService
{
    Task<ServiceResult<OperationLogListResult>> ListAdminAsync(OperationLogQuery query, DateTime? startTime, DateTime? endTime, CancellationToken cancellationToken);
    Task<ServiceResult<OperationLogListResult>> ListUserAsync(long userId, OperationLogQuery query, DateTime? startTime, DateTime? endTime, CancellationToken cancellationToken);
    Task WriteAsync(OperationLogWriteRequest request, CancellationToken cancellationToken);
}

public sealed class OperationLogWriteRequest
{
    public int? UserId { get; set; }
    public string? Type { get; set; }
    public string? Action { get; set; }
    public string? Content { get; set; }
    public string? Diff { get; set; }
    public string? Ip { get; set; }
    public string? Process { get; set; }
}

public sealed class OperationLogService : IOperationLogService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly ISqlSugarClient _db;

    public OperationLogService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<OperationLogListResult>> ListAdminAsync(
        OperationLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        CancellationToken cancellationToken)
    {
        query ??= new OperationLogQuery();
        var (page, pageSize) = ResolvePaging(query);
        var keyword = query.Keyword?.Trim();

        var q = _db.Queryable<OpLog>();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var userIds = await _db.Queryable<User>()
                .Where(u => SqlFunc.Contains(u.Name, keyword!))
                .Select(u => u.Id)
                .ToListAsync();

            if (userIds.Count > 0)
            {
                q = q.Where(log =>
                    SqlFunc.Contains(log.Action, keyword!) ||
                    SqlFunc.Contains(log.Content, keyword!) ||
                    SqlFunc.Contains(log.Ip, keyword!) ||
                    userIds.Contains(log.Uid ?? 0));
            }
            else
            {
                q = q.Where(log =>
                    SqlFunc.Contains(log.Action, keyword!) ||
                    SqlFunc.Contains(log.Content, keyword!) ||
                    SqlFunc.Contains(log.Ip, keyword!));
            }
        }

        if (startTime.HasValue && endTime.HasValue)
        {
            var start = startTime.Value;
            var end = endTime.Value;
            q = q.Where(log => log.CreateAt >= start && log.CreateAt <= end);
        }

        var total = await q.CountAsync();
        var rows = await q
            .OrderBy("id desc")
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var userMap = await LoadUserMapAsync(rows);
        var items = rows.Select(row => BuildItem(row, userMap)).ToList();
        return ServiceResult<OperationLogListResult>.Ok(new OperationLogListResult(items, total));
    }

    public async Task<ServiceResult<OperationLogListResult>> ListUserAsync(
        long userId,
        OperationLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<OperationLogListResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        query ??= new OperationLogQuery();
        var (page, pageSize) = ResolvePaging(query);
        var keyword = query.Keyword?.Trim();

        var q = _db.Queryable<OpLog>().Where(log => log.Uid == userId);
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            q = q.Where(log =>
                SqlFunc.Contains(log.Action, keyword!) ||
                SqlFunc.Contains(log.Content, keyword!) ||
                SqlFunc.Contains(log.Ip, keyword!));
        }

        if (startTime.HasValue && endTime.HasValue)
        {
            var start = startTime.Value;
            var end = endTime.Value;
            q = q.Where(log => log.CreateAt >= start && log.CreateAt <= end);
        }

        var total = await q.CountAsync();
        var rows = await q
            .OrderBy("id desc")
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var userMap = await LoadUserMapAsync(rows);
        var items = rows.Select(row => BuildItem(row, userMap)).ToList();
        return ServiceResult<OperationLogListResult>.Ok(new OperationLogListResult(items, total));
    }

    public async Task WriteAsync(OperationLogWriteRequest request, CancellationToken cancellationToken)
    {
        if (request == null || string.IsNullOrWhiteSpace(request.Action))
        {
            return;
        }

        var userId = request.UserId.GetValueOrDefault();
        var row = new OpLog
        {
            Uid = userId > 0 ? userId : null,
            Type = NormalizeOrDefault(request.Type, "system"),
            Action = NormalizeOrDefault(request.Action, "unknown"),
            Content = NormalizeOrDefault(request.Content, string.Empty),
            Diff = NormalizeOrDefault(request.Diff, string.Empty),
            Ip = NormalizeOrDefault(request.Ip, string.Empty),
            Process = NormalizeOrDefault(request.Process, string.Empty),
            CreateAt = DateTime.Now
        };

        try
        {
            await _db.Insertable(row).ExecuteCommandAsync();
            return;
        }
        catch
        {
            // Fallback to raw SQL for mixed legacy schemas.
        }

        try
        {
            await _db.Ado.ExecuteCommandAsync(
                "INSERT INTO op_log (uid, type, action, content, diff, ip, create_at, process) VALUES (@uid, @type, @action, @content, @diff, @ip, @create_at, @process)",
                new[]
                {
                    new SugarParameter("@uid", row.Uid),
                    new SugarParameter("@type", row.Type ?? string.Empty),
                    new SugarParameter("@action", row.Action ?? string.Empty),
                    new SugarParameter("@content", row.Content ?? string.Empty),
                    new SugarParameter("@diff", row.Diff ?? string.Empty),
                    new SugarParameter("@ip", row.Ip ?? string.Empty),
                    new SugarParameter("@create_at", row.CreateAt ?? DateTime.Now),
                    new SugarParameter("@process", row.Process ?? string.Empty)
                });
        }
        catch
        {
            // Operation logs are best-effort and should not break business flow.
        }
    }

    private static (int Page, int PageSize) ResolvePaging(OperationLogQuery query)
    {
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        return (page, pageSize);
    }

    private async Task<Dictionary<int, string?>> LoadUserMapAsync(IReadOnlyList<OpLog> rows)
    {
        var userIds = rows.Select(row => row.Uid ?? 0)
            .Where(id => id > 0)
            .Distinct()
            .ToList();
        if (userIds.Count == 0)
        {
            return new Dictionary<int, string?>();
        }

        var users = await _db.Queryable<User>()
            .Where(u => userIds.Contains(u.Id))
            .Select(u => new { u.Id, u.Name })
            .ToListAsync();

        var map = new Dictionary<int, string?>();
        foreach (var user in users)
        {
            map[user.Id] = user.Name;
        }

        return map;
    }

    private static OperationLogItem BuildItem(OpLog row, Dictionary<int, string?> userMap)
    {
        return new OperationLogItem
        {
            Id = row.Id,
            UserId = row.Uid ?? 0,
            Type = row.Type,
            Action = row.Action,
            Content = row.Content,
            Diff = row.Diff,
            Ip = row.Ip,
            Process = row.Process,
            Description = row.Content,
            CreatedAt = row.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss"),
            Username = userMap.TryGetValue(row.Uid ?? 0, out var name) ? name : null
        };
    }

    private static string? NormalizeOrNull(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return null;
        }

        return value.Trim();
    }

    private static string NormalizeOrDefault(string? value, string fallback)
    {
        var normalized = NormalizeOrNull(value);
        if (string.IsNullOrWhiteSpace(normalized))
        {
            return fallback;
        }

        return normalized;
    }
}
