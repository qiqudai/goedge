using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class ForwardGroupService : IForwardGroupService
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public ForwardGroupService(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public async Task<ServiceResult<ForwardGroupListResult>> ListAsync(
        string? keyword,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        var q = _db.Queryable<StreamGroup>();
        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<ForwardGroupListResult>.Fail(ErrorCodes.PermissionDenied);
            }
            q = q.Where(g => g.Uid == (int)userId.Value);
        }
        else if (userId is > 0)
        {
            q = q.Where(g => g.Uid == (int)userId.Value);
        }

        if (!string.IsNullOrWhiteSpace(keyword))
        {
            q = q.Where(g => SqlFunc.Contains(g.Name, keyword.Trim()));
        }

        var list = await q.OrderBy(g => g.Id, OrderByType.Desc).ToListAsync();
        var items = list.Select(g => new ForwardGroupDto
        {
            Id = g.Id,
            UserId = g.Uid ?? 0,
            Name = g.Name,
            Remark = g.Des
        }).ToList();

        return ServiceResult<ForwardGroupListResult>.Ok(new ForwardGroupListResult(items));
    }

    public async Task<ServiceResult<ForwardGroupDto>> CreateAsync(
        ForwardGroupUpsertRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        var targetUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (targetUserId <= 0)
        {
            return ServiceResult<ForwardGroupDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        if (string.IsNullOrWhiteSpace(request.Name))
        {
            return ServiceResult<ForwardGroupDto>.Fail(ErrorCodes.MissingParam, "forward_group_name_required");
        }

        var now = DateTime.Now;
        var group = new StreamGroup
        {
            Uid = (int)targetUserId,
            Name = request.Name?.Trim(),
            Des = request.Remark?.Trim()
        };

        var id = await _db.Insertable(group).ExecuteReturnIdentityAsync();
        group.Id = id;
        await _configVersionService.BumpAsync("forward_group", new[] { (long)id }, cancellationToken);

        return ServiceResult<ForwardGroupDto>.Ok(new ForwardGroupDto
        {
            Id = group.Id,
            UserId = group.Uid ?? 0,
            Name = group.Name,
            Remark = group.Des
        });
    }

    public async Task<ServiceResult<bool>> UpdateAsync(
        ForwardGroupUpsertRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (request.Id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "forward_group_id_required");
        }

        var remark = request.Remark?.Trim();
        var q = _db.Updateable<StreamGroup>().Where(g => g.Id == request.Id);
        if (!string.IsNullOrWhiteSpace(request.Name))
        {
            var name = request.Name!.Trim();
            q = q.SetColumns(g => new StreamGroup { Name = name, Des = remark });
        }
        else
        {
            q = q.SetColumns(g => new StreamGroup { Des = remark });
        }

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            q = q.Where(g => g.Uid == (int)userId.Value);
        }

        await q.ExecuteCommandAsync();
        await _configVersionService.BumpAsync("forward_group", new[] { request.Id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "forward_group_id_required");
        }

        var q = _db.Queryable<StreamGroup>().Where(g => g.Id == id);
        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
            }
            q = q.Where(g => g.Uid == (int)userId.Value);
        }

        var group = await q.FirstAsync();
        if (group == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "not_found");
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<MergeStreamGroup>()
                .Where(x => x.GroupId == group.Id)
                .ExecuteCommandAsync();

            await _db.Deleteable<StreamGroup>()
                .Where(x => x.Id == group.Id)
                .ExecuteCommandAsync();
        });

        await _configVersionService.BumpAsync("forward_group", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private static long ResolveUserId(long requestUserId, long? userId, bool isAdmin)
    {
        if (!isAdmin)
        {
            return userId ?? 0;
        }

        return requestUserId > 0 ? requestUserId : (userId ?? 0);
    }
}
