using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class SecurityRuleDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public SecurityRuleDeleteTaskHandler(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public string TaskType => AsyncTaskTypes.SecurityRuleDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("security rule delete payload is missing resource_id");
        }

        var ruleId = (int)payload.ResourceId;
        var rule = await _db.Queryable<CcRule>()
            .Where(x => x.Id == ruleId)
            .FirstAsync();
        if (rule == null)
        {
            return;
        }

        var refs = await SiteRuleUsageInspector.FindCcRuleUsagesAsync(_db, ruleId, cancellationToken);
        if (refs.Count > 0)
        {
            throw new InvalidOperationException("security rule is still referenced by sites");
        }

        await _db.Deleteable<CcRule>()
            .Where(x => x.Id == ruleId)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("cc_rule", new[] { payload.ResourceId }, cancellationToken);
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
