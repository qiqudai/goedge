using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using TaskEntity = Cnn.Domain.Entities.Task;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public interface IBackupLogService
{
    Task<ServiceResult<BackupLogListResult>> ListAsync(BackupLogQuery query, DateTime? startTime, DateTime? endTime, CancellationToken cancellationToken);
}

public sealed class BackupLogService : IBackupLogService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly ISqlSugarClient _db;

    public BackupLogService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<BackupLogListResult>> ListAsync(
        BackupLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        CancellationToken cancellationToken)
    {
        query ??= new BackupLogQuery();
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        var keyword = query.Keyword?.Trim();
        var q = _db.Queryable<TaskEntity>().Where(t => t.Type == "backup");

        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var normalized = keyword.Trim().ToLowerInvariant();
            if (normalized is "1" or "success" or "ok" or "done")
            {
                q = q.Where(t => t.State == "done");
            }
            else if (normalized is "0" or "fail" or "failed" or "error")
            {
                q = q.Where(t => t.State == "fail");
            }
            else
            {
                q = q.Where(t => SqlFunc.Contains(t.State, keyword!) || SqlFunc.Contains(t.Ret, keyword!));
            }
        }

        if (startTime.HasValue && endTime.HasValue)
        {
            var start = startTime.Value;
            var end = endTime.Value;
            q = q.Where(t => t.CreateAt >= start && t.CreateAt <= end);
        }

        var total = await q.CountAsync();
        var rows = await q
            .OrderBy("id desc")
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var items = rows.Select(row =>
        {
            var finishedAt = row.EndAt ?? row.StartAt;
            var status = string.Equals(row.State, "done", StringComparison.OrdinalIgnoreCase) ? 1 : 0;
            return new BackupLogItem
            {
                Id = row.Id,
                CreatedAt = row.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss"),
                FinishedAt = finishedAt?.ToString("yyyy-MM-dd HH:mm:ss"),
                Status = status,
                Result = row.Ret
            };
        }).ToList();

        return ServiceResult<BackupLogListResult>.Ok(new BackupLogListResult(items, total));
    }
}
