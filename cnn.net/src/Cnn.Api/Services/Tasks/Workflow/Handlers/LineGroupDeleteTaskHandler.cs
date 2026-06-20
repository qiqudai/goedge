using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;
using StreamEntity = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class LineGroupDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public LineGroupDeleteTaskHandler(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public string TaskType => AsyncTaskTypes.LineGroupDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("line group delete payload is missing resource_id");
        }

        var groupId = (int)payload.ResourceId;
        var group = await _db.Queryable<NodeGroup>()
            .Where(x => x.Id == groupId)
            .FirstAsync();
        if (group == null)
        {
            return;
        }

        var lineRefs = await _db.Queryable<Line>()
            .Where(x => x.NodeGroupId == groupId)
            .CountAsync();
        if (lineRefs > 0)
        {
            throw new InvalidOperationException("line group is still referenced by lines");
        }

        var packageRefs = await _db.Queryable<Package>()
            .Where(x => x.NodeGroupId == groupId || x.BackupNodeGroup == groupId)
            .CountAsync();
        if (packageRefs > 0)
        {
            throw new InvalidOperationException("line group is still referenced by packages");
        }

        var siteRefs = await _db.Queryable<Site>()
            .Where(x => x.NodeGroupId == groupId || x.BackupNodeGroup == groupId)
            .CountAsync();
        if (siteRefs > 0)
        {
            throw new InvalidOperationException("line group is still referenced by sites");
        }

        var streamRefs = await _db.Queryable<StreamEntity>()
            .Where(x => x.NodeGroupId == groupId || x.BackupNodeGroup == groupId)
            .CountAsync();
        if (streamRefs > 0)
        {
            throw new InvalidOperationException("line group is still referenced by streams");
        }

        await _db.Deleteable<NodeGroup>()
            .Where(x => x.Id == groupId)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("node_group", new[] { payload.ResourceId }, cancellationToken);
    }

    private static DeletePayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new DeletePayload();
        }

        try
        {
            return JsonSerializer.Deserialize<DeletePayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                   ?? new DeletePayload();
        }
        catch
        {
            return new DeletePayload();
        }
    }

    private sealed class DeletePayload
    {
        [JsonPropertyName("resource_id")]
        public long ResourceId { get; init; }
    }
}
