using System.Text.Json;
using System.Text.Json.Serialization;
using SqlSugar;
using Stream = Cnn.Domain.Entities.Stream;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class StreamStatusTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public StreamStatusTaskHandler(
        ISqlSugarClient db,
        IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public string TaskType => throw new NotSupportedException("Resolve task type via CanHandle.");

    public bool CanHandle(string taskType)
    {
        return string.Equals(taskType, AsyncTaskTypes.StreamEnable, StringComparison.OrdinalIgnoreCase)
               || string.Equals(taskType, AsyncTaskTypes.StreamDisable, StringComparison.OrdinalIgnoreCase);
    }

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        var streamIds = NormalizeIds(payload.ResourceIds);
        if (streamIds.Count == 0 || payload.Enable == null)
        {
            throw new InvalidOperationException("stream status payload is invalid");
        }

        await _db.Updateable<Stream>()
            .SetColumns(s => new Stream
            {
                Enable = payload.Enable,
                State = payload.Enable.Value ? "running" : "stop"
            })
            .Where(s => streamIds.Contains(s.Id))
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("forward", streamIds, cancellationToken);
    }

    private static StreamStatusPayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new StreamStatusPayload();
        }

        try
        {
            return JsonSerializer.Deserialize<StreamStatusPayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                   ?? new StreamStatusPayload();
        }
        catch
        {
            return new StreamStatusPayload();
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

    private sealed class StreamStatusPayload
    {
        [JsonPropertyName("resource_ids")]
        public IReadOnlyList<long>? ResourceIds { get; init; }
        [JsonPropertyName("enable")]
        public bool? Enable { get; init; }
    }
}
