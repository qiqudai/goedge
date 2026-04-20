using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class SiteGroupDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public SiteGroupDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.SiteGroup;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "站点分组 ID 无效");
        }

        var refs = await _db.Ado.SqlQueryAsync<SiteGroupReference>(
            """
SELECT
  s.id,
  s.domain AS primary_domain,
  u.name AS username
FROM merge_site_group m
JOIN site s ON s.id = m.site_id
JOIN `user` u ON u.id = s.uid
WHERE m.group_id = @siteGroupId
ORDER BY s.id ASC
""",
            new { siteGroupId = resourceId });

        return new DeleteGuardResult
        {
            CanDelete = true,
            Message = refs.Count == 0
                ? "站点分组可直接删除。"
                : "站点分组可直接删除，删除后将自动清理该分组与站点的关联关系。",
            References = refs.Select(r => new DeleteReferenceItem
            {
                ResourceType = ResourceTypes.Site,
                ResourceId = r.Id,
                DisplayName = $"{r.Username} / {r.PrimaryDomain}",
                Relation = "merge_site_group"
            }).ToList()
        };
    }

    private sealed class SiteGroupReference
    {
        public long Id { get; init; }
        public string PrimaryDomain { get; init; } = string.Empty;
        public string Username { get; init; } = string.Empty;
    }
}
