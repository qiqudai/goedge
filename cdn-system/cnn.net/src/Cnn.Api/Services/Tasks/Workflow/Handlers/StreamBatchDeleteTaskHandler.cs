using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;
using StreamEntity = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class StreamBatchDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public StreamBatchDeleteTaskHandler(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public string TaskType => AsyncTaskTypes.StreamBatchDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        var streamIds = NormalizeIds(payload.ResourceIds);
        if (streamIds.Count == 0)
        {
            throw new InvalidOperationException("stream batch delete payload is missing resource_ids");
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<MergeStreamGroup>()
                .Where(x => x.StreamId.HasValue && streamIds.Contains(x.StreamId.Value))
                .ExecuteCommandAsync();

            await _db.Deleteable<StreamEntity>()
                .Where(x => streamIds.Contains(x.Id))
                .ExecuteCommandAsync();
        });

        await _configVersionService.BumpAsync("forward", streamIds, cancellationToken);
    }

    private static BatchDeletePayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new BatchDeletePayload();
        }

        try
        {
            return JsonSerializer.Deserialize<BatchDeletePayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                   ?? new BatchDeletePayload();
        }
        catch
        {
            return new BatchDeletePayload();
        }
    }

    private static List<long> NormalizeIds(IReadOnlyList<long>? ids)
    {
        return ids?
            .Where(id => id > 0)
            .Distinct()
            .OrderBy(id => id)
            .ToList()
               ?? new List<long>();
    }

    private sealed class BatchDeletePayload
    {
        [JsonPropertyName("resource_ids")]
        public IReadOnlyList<long>? ResourceIds { get; init; }
    }
}
