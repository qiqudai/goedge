using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class NodeDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;

    public NodeDeleteTaskHandler(ISqlSugarClient db)
    {
        _db = db;
    }

    public string TaskType => AsyncTaskTypes.NodeDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("node delete payload is missing resource_id");
        }

        var nodeId = (int)payload.ResourceId;
        var exists = await _db.Queryable<Node>().AnyAsync(n => n.Id == nodeId);
        if (!exists)
        {
            return;
        }

        var lineCount = await _db.Queryable<Line>()
            .Where(l => l.NodeId == nodeId || l.NodeIpId == nodeId)
            .CountAsync();
        if (lineCount > 0)
        {
            throw new InvalidOperationException("node is still referenced by line groups");
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<Node>().Where(n => n.Pid == nodeId).ExecuteCommandAsync();
            await _db.Deleteable<Node>().Where(n => n.Id == nodeId).ExecuteCommandAsync();
        });
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
