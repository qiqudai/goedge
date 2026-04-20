using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class AnnouncementService : IAnnouncementService
{
    private const string AnnouncementType = "announcement";
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly ISqlSugarClient _db;

    public AnnouncementService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<AnnouncementListResult>> ListAsync(AnnouncementListQuery query, CancellationToken cancellationToken)
    {
        query ??= new AnnouncementListQuery();
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        var q = _db.Queryable<Message>().Where(m => m.Type == AnnouncementType);
        var keyword = query.Keyword?.Trim();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            q = q.Where(m => SqlFunc.Contains(m.Title, keyword!) || SqlFunc.Contains(m.Content, keyword!));
        }

        var total = await q.CountAsync();
        var list = await q.OrderBy(m => m.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var items = list.Select(item => new AnnouncementItemDto
        {
            Id = item.Id,
            Title = item.Title,
            Content = item.Content,
            IsShow = item.IsShow ?? false,
            IsRed = item.IsRed ?? false,
            IsBold = item.IsBold ?? false,
            CreatedAt = item.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss")
        }).ToList();

        return ServiceResult<AnnouncementListResult>.Ok(new AnnouncementListResult(items, total));
    }

    public async Task<ServiceResult<bool>> CreateAsync(AnnouncementUpsertRequest request, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var now = DateTime.Now;
        var item = new Message
        {
            Type = AnnouncementType,
            Receive = 0,
            Title = request.Title,
            Content = request.Content,
            IsShow = request.IsShow ?? false,
            IsRed = request.IsRed ?? false,
            IsBold = request.IsBold ?? false,
            CreateAt = now,
            UpdateAt = now
        };

        await _db.Insertable(item).ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long id, AnnouncementUpsertRequest request, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        if (request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var now = DateTime.Now;
        await _db.Updateable<Message>()
            .SetColumns(m => new Message
            {
                Title = request.Title,
                Content = request.Content,
                IsShow = request.IsShow ?? false,
                IsRed = request.IsRed ?? false,
                IsBold = request.IsBold ?? false,
                UpdateAt = now
            })
            .Where(m => m.Id == id)
            .ExecuteCommandAsync();

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        await _db.Deleteable<Message>().Where(m => m.Id == id).ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }
}
