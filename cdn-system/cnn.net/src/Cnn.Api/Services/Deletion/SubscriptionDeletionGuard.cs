using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class SubscriptionDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public SubscriptionDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.Subscription;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "已售套餐 ID 无效");
        }

        var refs = new List<DeleteReferenceItem>();

        var siteRefs = await _db.Ado.SqlQueryAsync<SiteReference>(
            """
SELECT s.id, s.domain AS primary_domain, u.name AS username
FROM site s
JOIN `user` u ON u.id = s.uid
WHERE s.user_package = @subscriptionId
ORDER BY s.id ASC
""",
            new { subscriptionId = resourceId });

        refs.AddRange(siteRefs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.Site,
            ResourceId = r.Id,
            DisplayName = $"{r.Username} / {r.PrimaryDomain}",
            Relation = "site.user_package"
        }));

        var streamRefs = await _db.Ado.SqlQueryAsync<StreamReference>(
            """
SELECT
  s.id,
  COALESCE(s.record_id, CAST(s.id AS CHAR)) AS name,
  s.listen AS listen_port,
  u.name AS username
FROM stream s
JOIN `user` u ON u.id = s.uid
WHERE s.user_package = @subscriptionId
ORDER BY s.id ASC
""",
            new { subscriptionId = resourceId });

        refs.AddRange(streamRefs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.StreamApp,
            ResourceId = r.Id,
            DisplayName = $"{r.Username} / {r.Name} / tcp:{r.ListenPort}",
            Relation = "stream.user_package"
        }));

        if (refs.Count == 0)
        {
            return DeleteGuardResult.Allow();
        }

        return DeleteGuardResult.Deny(
            "SUBSCRIPTION_IN_USE",
            "已售套餐仍被站点或四层转发使用，请先迁移或删除相关资源后再删除。",
            refs);
    }

    private sealed class SiteReference
    {
        public long Id { get; init; }
        public string PrimaryDomain { get; init; } = string.Empty;
        public string Username { get; init; } = string.Empty;
    }

    private sealed class StreamReference
    {
        public long Id { get; init; }
        public string Name { get; init; } = string.Empty;
        public string ListenPort { get; init; } = string.Empty;
        public string Username { get; init; } = string.Empty;
    }
}
