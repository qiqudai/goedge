using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public interface IMailLogService
{
    Task<ServiceResult<MailLogListResult>> ListAsync(MailLogQuery query, DateTime? startTime, DateTime? endTime, CancellationToken cancellationToken);
}

public sealed class MailLogService : IMailLogService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly ISqlSugarClient _db;

    public MailLogService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<MailLogListResult>> ListAsync(
        MailLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        CancellationToken cancellationToken)
    {
        query ??= new MailLogQuery();
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        var keyword = query.Keyword?.Trim();
        var q = _db.Queryable<Message>();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            q = q.Where(m => SqlFunc.Contains(m.Title, keyword!) || SqlFunc.Contains(m.Content, keyword!));
            if (long.TryParse(keyword, out var id))
            {
                q = q.Where(m => m.Id == id || m.Receive == id);
            }
        }

        if (startTime.HasValue && endTime.HasValue)
        {
            var start = startTime.Value;
            var end = endTime.Value;
            q = q.Where(m => m.CreateAt >= start && m.CreateAt <= end);
        }

        var total = await q.CountAsync();
        var rows = await q
            .OrderBy("id desc")
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var items = rows.Select(row =>
        {
            var medium = row.PhoneNeedSend == true ? "SMS" : "Email";
            var status = row.EmailIsSent == true || row.PhoneIsSent == true ? 1 : 0;
            return new MailLogItem
            {
                MessageId = row.Id,
                UserId = row.Receive ?? 0,
                Subject = row.Title,
                Medium = medium,
                Fails = 0,
                Status = status,
                Reason = string.Empty,
                CreatedAt = row.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss")
            };
        }).ToList();

        return ServiceResult<MailLogListResult>.Ok(new MailLogListResult(items, total));
    }
}
