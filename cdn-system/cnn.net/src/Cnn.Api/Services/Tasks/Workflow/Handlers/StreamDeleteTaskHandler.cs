using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Common;
using SqlSugar;
using Task = System.Threading.Tasks.Task;
using StreamEntity = Cnn.Domain.Entities.Stream;
using Cnn.Domain.Entities;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class StreamDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public StreamDeleteTaskHandler(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public string TaskType => AsyncTaskTypes.StreamDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("stream delete payload is missing resource_id");
        }

        var streamId = (int)payload.ResourceId;
        var stream = await _db.Queryable<StreamEntity>()
            .Where(x => x.Id == streamId)
            .FirstAsync();
        if (stream == null)
        {
            return;
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<MergeStreamGroup>()
                .Where(x => x.StreamId == streamId)
                .ExecuteCommandAsync();

            await _db.Deleteable<StreamEntity>()
                .Where(x => x.Id == streamId)
                .ExecuteCommandAsync();
        });

        await _configVersionService.BumpAsync("forward", new[] { payload.ResourceId }, cancellationToken);
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
