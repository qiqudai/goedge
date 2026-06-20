using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public interface ILoginLogService
{
    Task<ServiceResult<LoginLogListResult>> ListAsync(LoginLogQuery query, DateTime? startTime, DateTime? endTime, CancellationToken cancellationToken);
}

public sealed class LoginLogService : ILoginLogService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly ISqlSugarClient _db;

    public LoginLogService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<LoginLogListResult>> ListAsync(
        LoginLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        CancellationToken cancellationToken)
    {
        query ??= new LoginLogQuery();
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        var keyword = query.Keyword?.Trim();

        var q = _db.Queryable<LoginLog>();
        List<int> matchedUserIds = new();

        if (!string.IsNullOrWhiteSpace(keyword))
        {
            matchedUserIds = await _db.Queryable<User>()
                .Where(u => SqlFunc.Contains(u.Name, keyword!))
                .Select(u => u.Id)
                .ToListAsync();

            if (matchedUserIds.Count > 0)
            {
                q = q.Where(log => SqlFunc.Contains(log.Ip, keyword!) || matchedUserIds.Contains(log.Uid ?? 0));
            }
            else
            {
                q = q.Where(log => SqlFunc.Contains(log.Ip, keyword!));
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

        var userIds = rows.Select(row => row.Uid ?? 0)
            .Where(id => id > 0)
            .Distinct()
            .ToList();

        var userMap = new Dictionary<int, string?>();
        if (userIds.Count > 0)
        {
            var users = await _db.Queryable<User>()
                .Where(u => userIds.Contains(u.Id))
                .Select(u => new { u.Id, u.Name })
                .ToListAsync();

            foreach (var user in users)
            {
                userMap[user.Id] = user.Name;
            }
        }

        var items = rows.Select(row => new LoginLogItem
        {
            Id = row.Id,
            UserId = row.Uid ?? 0,
            Username = userMap.TryGetValue(row.Uid ?? 0, out var name) ? name : null,
            Ip = row.Ip,
            Success = row.Success ?? false,
            PostContent = row.PostContent,
            CreatedAt = row.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss")
        }).ToList();

        return ServiceResult<LoginLogListResult>.Ok(new LoginLogListResult(items, total));
    }
}
