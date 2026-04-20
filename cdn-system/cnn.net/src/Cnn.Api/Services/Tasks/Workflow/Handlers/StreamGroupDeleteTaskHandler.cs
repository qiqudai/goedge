using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class StreamGroupDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;

    public StreamGroupDeleteTaskHandler(ISqlSugarClient db)
    {
        _db = db;
    }

    public string TaskType => AsyncTaskTypes.StreamGroupDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("stream group delete payload is missing resource_id");
        }

        var groupId = (int)payload.ResourceId;
        var group = await _db.Queryable<StreamGroup>()
            .Where(x => x.Id == groupId)
            .FirstAsync();
        if (group == null)
        {
            return;
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<MergeStreamGroup>()
                .Where(x => x.GroupId == groupId)
                .ExecuteCommandAsync();

            await _db.Deleteable<StreamGroup>()
                .Where(x => x.Id == groupId)
                .ExecuteCommandAsync();
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
