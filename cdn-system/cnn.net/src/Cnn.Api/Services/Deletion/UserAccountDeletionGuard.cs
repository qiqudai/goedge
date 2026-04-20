using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class UserAccountDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public UserAccountDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.UserAccount;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "用户 ID 无效");
        }

        var exists = await _db.Queryable<User>()
            .Where(x => x.Id == resourceId)
            .AnyAsync();

        if (!exists)
        {
            return DeleteGuardResult.Deny("USER_NOT_FOUND", "用户不存在，无法删除。");
        }

        return DeleteGuardResult.Allow();
    }
}
