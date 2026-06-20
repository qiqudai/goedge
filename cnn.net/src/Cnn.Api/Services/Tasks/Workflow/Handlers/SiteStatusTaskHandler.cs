using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class SiteStatusTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public SiteStatusTaskHandler(
        ISqlSugarClient db,
        IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public string TaskType => throw new NotSupportedException("Resolve task type via CanHandle.");

    public bool CanHandle(string taskType)
    {
        return string.Equals(taskType, AsyncTaskTypes.SiteEnable, StringComparison.OrdinalIgnoreCase)
               || string.Equals(taskType, AsyncTaskTypes.SiteDisable, StringComparison.OrdinalIgnoreCase);
    }

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        var siteIds = NormalizeIds(payload.ResourceIds);
        if (siteIds.Count == 0 || payload.Enable == null)
        {
            throw new InvalidOperationException("site status payload is invalid");
        }

        await _db.Updateable<Site>()
            .SetColumns(s => new Site
            {
                Enable = payload.Enable,
                State = payload.Enable.Value ? "running" : "stop"
            })
            .Where(s => siteIds.Contains(s.Id))
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("site", siteIds, cancellationToken);
    }

    private static SiteStatusPayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new SiteStatusPayload();
        }

        try
        {
            return JsonSerializer.Deserialize<SiteStatusPayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                   ?? new SiteStatusPayload();
        }
        catch
        {
            return new SiteStatusPayload();
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

    private sealed class SiteStatusPayload
    {
        [JsonPropertyName("resource_ids")]
        public IReadOnlyList<long>? ResourceIds { get; init; }
        [JsonPropertyName("enable")]
        public bool? Enable { get; init; }
    }
}
