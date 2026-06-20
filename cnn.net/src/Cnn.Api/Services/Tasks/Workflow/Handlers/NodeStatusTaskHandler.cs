using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class NodeStatusTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IDnsSyncService _dnsSyncService;
    private readonly IAdminEventPublisher _eventPublisher;

    public NodeStatusTaskHandler(
        ISqlSugarClient db,
        IDnsSyncService dnsSyncService,
        IAdminEventPublisher eventPublisher)
    {
        _db = db;
        _dnsSyncService = dnsSyncService;
        _eventPublisher = eventPublisher;
    }

    public string TaskType => throw new NotSupportedException("Resolve task type via CanHandle.");

    public bool CanHandle(string taskType)
    {
        return string.Equals(taskType, AsyncTaskTypes.NodeEnable, StringComparison.OrdinalIgnoreCase)
               || string.Equals(taskType, AsyncTaskTypes.NodeDisable, StringComparison.OrdinalIgnoreCase);
    }

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0 || payload.Enable == null)
        {
            throw new InvalidOperationException("node status payload is invalid");
        }

        var nodeId = (int)payload.ResourceId;
        var exists = await _db.Queryable<Node>()
            .Where(x => x.Id == nodeId)
            .AnyAsync();
        if (!exists)
        {
            return;
        }

        var syncTask = payload.Enable.Value ? "sync_enable" : "sync_disable";
        var now = DateTime.Now;

        await _db.Updateable<Node>()
            .SetColumns(x => new Node
            {
                Enable = payload.Enable,
                ConfigTask = syncTask,
                UpdateAt = now
            })
            .Where(x => x.Id == nodeId)
            .ExecuteCommandAsync();

        await _db.Updateable<Node>()
            .SetColumns(x => new Node
            {
                Enable = payload.Enable,
                UpdateAt = now
            })
            .Where(x => x.Pid == nodeId)
            .ExecuteCommandAsync();

        await _dnsSyncService.SyncPackageCnameForNodesAsync(new[] { payload.ResourceId }, payload.Enable.Value ? "add" : "delete");
        await _eventPublisher.PublishToAdminsAsync("node.status.changed", new
        {
            node_id = payload.ResourceId,
            enable = payload.Enable,
            online = false,
            checked_at = DateTimeOffset.UtcNow.ToUnixTimeSeconds()
        });
    }

    private static NodeStatusPayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new NodeStatusPayload();
        }

        try
        {
            return JsonSerializer.Deserialize<NodeStatusPayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                   ?? new NodeStatusPayload();
        }
        catch
        {
            return new NodeStatusPayload();
        }
    }

    private sealed class NodeStatusPayload
    {
        [JsonPropertyName("resource_id")]
        public long ResourceId { get; init; }
        [JsonPropertyName("enable")]
        public bool? Enable { get; init; }
    }
}
