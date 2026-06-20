using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class NodeDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public NodeDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.Node;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "节点 ID 无效");
        }

        var refs = await _db.Ado.SqlQueryAsync<NodeDeletionReference>(
            """
SELECT
  ng.id AS line_group_id,
  ng.name AS line_group_name,
  l.line_id AS line_code,
  l.line_name
FROM line l
JOIN node_group ng ON ng.id = l.node_group_id
WHERE l.node_id = @nodeId OR l.node_ip_id = @nodeId
ORDER BY ng.id ASC, l.line_id ASC
""",
            new { nodeId = resourceId });

        if (refs.Count == 0)
        {
            return DeleteGuardResult.Allow();
        }

        var items = refs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.LineGroup,
            ResourceId = r.LineGroupId,
            DisplayName = $"{r.LineGroupName} / {r.LineCode}",
            Relation = "line"
        }).ToList();

        return DeleteGuardResult.Deny(
            "NODE_IN_USE",
            "节点正在被线路组引用，请先从线路组中移除该节点后再删除。",
            items);
    }

    private sealed class NodeDeletionReference
    {
        public long LineGroupId { get; init; }
        public string LineGroupName { get; init; } = string.Empty;
        public string LineCode { get; init; } = string.Empty;
        public string LineName { get; init; } = string.Empty;
    }
}
