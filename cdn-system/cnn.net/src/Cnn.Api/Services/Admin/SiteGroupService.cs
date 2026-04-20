using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class SiteGroupService : ISiteGroupService
{
    private readonly ISqlSugarClient _db;

    public SiteGroupService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<SiteGroupListResult>> ListAsync(
        SiteGroupListQuery query,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new SiteGroupListQuery();
        var page = query.Page.GetValueOrDefault(1);
        var pageSize = query.PageSize.GetValueOrDefault(10);
        if (page < 1) page = 1;
        if (pageSize < 1) pageSize = 10;

        var q = _db.Queryable<SiteGroup>();
        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<SiteGroupListResult>.Fail(ErrorCodes.PermissionDenied);
            }
            q = q.Where(g => g.Uid == (int)userId.Value);
        }
        else if (query.UserId is > 0)
        {
            q = q.Where(g => g.Uid == (int)query.UserId.Value);
        }

        if (!string.IsNullOrWhiteSpace(query.Keyword))
        {
            var keyword = query.Keyword.Trim();
            q = q.Where(g => SqlFunc.Contains(g.Name, keyword));
        }

        var total = await q.CountAsync();
        var list = await q.OrderBy(g => g.Id, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);

        var items = list.Select(g => new SiteGroupDto
        {
            Id = g.Id,
            UserId = g.Uid ?? 0,
            Name = g.Name,
            Remark = g.Des
        }).ToList();

        return ServiceResult<SiteGroupListResult>.Ok(new SiteGroupListResult(items, (int)total));
    }

    public async Task<ServiceResult<SiteGroupDto>> CreateAsync(
        SiteGroupUpsertRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        request ??= new SiteGroupUpsertRequest();
        var targetUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (targetUserId <= 0)
        {
            return ServiceResult<SiteGroupDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var group = new SiteGroup
        {
            Uid = (int)targetUserId,
            Name = request.Name?.Trim(),
            Des = request.Remark?.Trim()
        };

        var id = await _db.Insertable(group).ExecuteReturnIdentityAsync();
        group.Id = id;

        return ServiceResult<SiteGroupDto>.Ok(new SiteGroupDto
        {
            Id = group.Id,
            UserId = group.Uid ?? 0,
            Name = group.Name,
            Remark = group.Des
        });
    }

    public async Task<ServiceResult<bool>> UpdateAsync(
        long id,
        SiteGroupUpsertRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "site_group_id_required");
        }

        request ??= new SiteGroupUpsertRequest();
        var q = _db.Updateable<SiteGroup>().Where(g => g.Id == id);

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            q = q.Where(g => g.Uid == (int)userId.Value);
        }

        var name = request.Name?.Trim();
        var remark = request.Remark?.Trim();
        if (!string.IsNullOrWhiteSpace(name))
        {
            q = q.SetColumns(g => new SiteGroup { Name = name, Des = remark });
        }
        else
        {
            q = q.SetColumns(g => new SiteGroup { Des = remark });
        }

        await q.ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(
        long id,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "site_group_id_required");
        }

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }

            var owned = await _db.Queryable<SiteGroup>()
                .Where(g => g.Id == id && g.Uid == (int)userId.Value)
                .AnyAsync();
            if (!owned)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "not_found");
            }
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<MergeSiteGroup>().Where(r => r.GroupId == id).ExecuteCommandAsync();
            await _db.Deleteable<SiteGroup>().Where(g => g.Id == id).ExecuteCommandAsync();
        });

        return ServiceResult<bool>.Ok(true);
    }

    private static long ResolveUserId(long? requestUserId, long? userId, bool isAdmin)
    {
        if (!isAdmin)
        {
            return userId ?? 0;
        }

        if (requestUserId is > 0)
        {
            return requestUserId.Value;
        }

        return userId ?? 0;
    }
}
