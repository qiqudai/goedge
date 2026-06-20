using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class UserGroupService : IUserGroupService
{
    private readonly ISqlSugarClient _db;

    public UserGroupService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<UserGroupListResult>> ListAsync(CancellationToken cancellationToken)
    {
        var list = await _db.Queryable<UserGroup>()
            .OrderBy(x => x.Id, OrderByType.Asc)
            .ToListAsync();

        var items = list.Select(x => new UserGroupDto
        {
            Id = x.Id,
            Name = x.Name,
            Des = x.Des
        }).ToList();

        return ServiceResult<UserGroupListResult>.Ok(new UserGroupListResult(items));
    }

    public async Task<ServiceResult<UserGroupDto>> CreateAsync(UserGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        request ??= new UserGroupUpsertRequest();
        var name = request.Name?.Trim();
        var des = request.Des?.Trim();
        if (string.IsNullOrWhiteSpace(name))
        {
            return ServiceResult<UserGroupDto>.Fail(ErrorCodes.InvalidParam, "user_group_name_required");
        }

        var exists = await _db.Queryable<UserGroup>()
            .AnyAsync(x => x.Name == name);
        if (exists)
        {
            return ServiceResult<UserGroupDto>.Fail(ErrorCodes.AlreadyExists, "user_group_exists");
        }

        var entity = new UserGroup
        {
            Name = name,
            Des = des
        };
        var inserted = await _db.Insertable(entity).ExecuteReturnEntityAsync();
        if (inserted == null || inserted.Id <= 0)
        {
            return ServiceResult<UserGroupDto>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<UserGroupDto>.Ok(new UserGroupDto
        {
            Id = inserted.Id,
            Name = inserted.Name,
            Des = inserted.Des
        });
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_group_id_required");
        }

        var usingCount = await _db.Queryable<User>()
            .Where(x => x.GroupId == id)
            .CountAsync();
        if (usingCount > 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InUse, "user_group_in_use");
        }

        var rows = await _db.Deleteable<UserGroup>()
            .Where(x => x.Id == id)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "user_group_not_found");
        }

        return ServiceResult<bool>.Ok(true);
    }
}

