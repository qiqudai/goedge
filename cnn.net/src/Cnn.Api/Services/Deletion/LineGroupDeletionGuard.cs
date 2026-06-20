using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class LineGroupDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public LineGroupDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.LineGroup;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "线路组 ID 无效");
        }

        var refs = new List<DeleteReferenceItem>();

        var planRefs = await _db.Ado.SqlQueryAsync<PlanReference>(
            """
SELECT id, name
FROM package
WHERE node_group_id = @lineGroupId OR backup_node_group = @lineGroupId
ORDER BY id ASC
""",
            new { lineGroupId = resourceId });

        refs.AddRange(planRefs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.ProductPlan,
            ResourceId = r.Id,
            DisplayName = r.Name,
            Relation = "package.node_group"
        }));

        var siteRefs = await _db.Ado.SqlQueryAsync<SiteReference>(
            """
SELECT s.id, s.domain AS primary_domain, u.name AS username
FROM site s
JOIN `user` u ON u.id = s.uid
WHERE s.node_group_id = @lineGroupId OR s.backup_node_group = @lineGroupId
ORDER BY s.id ASC
""",
            new { lineGroupId = resourceId });

        refs.AddRange(siteRefs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.Site,
            ResourceId = r.Id,
            DisplayName = $"{r.Username} / {r.PrimaryDomain}",
            Relation = "site.node_group"
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
WHERE s.node_group_id = @lineGroupId OR s.backup_node_group = @lineGroupId
ORDER BY s.id ASC
""",
            new { lineGroupId = resourceId });

        refs.AddRange(streamRefs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.StreamApp,
            ResourceId = r.Id,
            DisplayName = $"{r.Username} / {r.Name} / tcp:{r.ListenPort}",
            Relation = "stream.node_group"
        }));

        if (refs.Count == 0)
        {
            return DeleteGuardResult.Allow();
        }

        return DeleteGuardResult.Deny(
            "LINE_GROUP_IN_USE",
            "线路组仍被套餐、站点或四层转发引用，无法删除。",
            refs);
    }

    private sealed class PlanReference
    {
        public long Id { get; init; }
        public string Name { get; init; } = string.Empty;
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
