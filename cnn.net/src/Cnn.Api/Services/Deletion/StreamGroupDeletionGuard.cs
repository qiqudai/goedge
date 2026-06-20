using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class StreamGroupDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public StreamGroupDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.StreamGroup;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "转发分组 ID 无效");
        }

        var refs = await _db.Ado.SqlQueryAsync<StreamGroupReference>(
            """
SELECT
  s.id,
  COALESCE(s.record_id, CAST(s.id AS CHAR)) AS name,
  s.listen AS listen_port,
  u.name AS username
FROM merge_stream_group m
JOIN stream s ON s.id = m.stream_id
JOIN `user` u ON u.id = s.uid
WHERE m.group_id = @groupId
ORDER BY s.id ASC
""",
            new { groupId = resourceId });

        return new DeleteGuardResult
        {
            CanDelete = true,
            Message = refs.Count == 0
                ? "转发分组可直接删除。"
                : "转发分组可直接删除，删除后将自动清理该分组与转发的关联关系。",
            References = refs.Select(r => new DeleteReferenceItem
            {
                ResourceType = ResourceTypes.StreamApp,
                ResourceId = r.Id,
                DisplayName = $"{r.Username} / {r.Name} / tcp:{r.ListenPort}",
                Relation = "merge_stream_group"
            }).ToList()
        };
    }

    private sealed class StreamGroupReference
    {
        public long Id { get; init; }
        public string Name { get; init; } = string.Empty;
        public string ListenPort { get; init; } = string.Empty;
        public string Username { get; init; } = string.Empty;
    }
}
