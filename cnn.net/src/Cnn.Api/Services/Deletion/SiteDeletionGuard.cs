using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class SiteDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public SiteDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.Site;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "站点 ID 无效");
        }

        var site = await _db.Queryable<Site>()
            .Where(x => x.Id == resourceId)
            .FirstAsync();
        if (site == null)
        {
            return DeleteGuardResult.Deny("SITE_NOT_FOUND", "站点不存在，无法删除。");
        }

        if (site.Enable == true)
        {
            return DeleteGuardResult.Deny("SITE_MUST_DISABLE_FIRST", "站点仍处于启用状态，请先禁用站点后再删除。");
        }

        return DeleteGuardResult.Allow();
    }
}
