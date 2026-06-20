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

        var resourceType = string.IsNullOrWhiteSpace(payload.ResourceType)
            ? ResourceTypes.SecurityRule
            : payload.ResourceType.ToLowerInvariant();

        switch (resourceType)
        {
            case ResourceTypes.SecurityRule:
            case ResourceTypes.CcRuleGroup:
                await DeleteCcRuleGroupAsync(payload.ResourceId, cancellationToken);
                break;

            case ResourceTypes.CcMatcher:
                await DeleteCcMatcherAsync(payload.ResourceId, cancellationToken);
                break;

            case ResourceTypes.CcFilter:
                await DeleteCcFilterAsync(payload.ResourceId, cancellationToken);
                break;

            default:
                throw new InvalidOperationException($"unsupported security rule resource_type '{payload.ResourceType}'");
        }
    }

    private async Task DeleteCcRuleGroupAsync(long resourceId, CancellationToken cancellationToken)
    {
        var ruleId = (int)resourceId;
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
            throw new InvalidOperationException("cc rule group is still referenced by sites");
        }

        await _db.Deleteable<CcRule>()
            .Where(x => x.Id == ruleId)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("cc_rule", new[] { resourceId }, cancellationToken);
    }

    private async Task DeleteCcMatcherAsync(long resourceId, CancellationToken cancellationToken)
    {
        var matcherId = (int)resourceId;
        var matcher = await _db.Queryable<CcMatch>()
            .Where(x => x.Id == matcherId)
            .FirstAsync();
        if (matcher == null)
        {
            return;
        }

        var refs = await SiteRuleUsageInspector.FindCcMatcherUsagesAsync(_db, matcherId, cancellationToken);
        if (refs.Count > 0)
        {
            throw new InvalidOperationException("cc matcher is still referenced by rule groups");
        }

        await _db.Deleteable<CcMatch>()
            .Where(x => x.Id == matcherId)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("cc_match", new[] { resourceId }, cancellationToken);
    }

    private async Task DeleteCcFilterAsync(long resourceId, CancellationToken cancellationToken)
    {
        var filterId = (int)resourceId;
        var filter = await _db.Queryable<CcFilter>()
            .Where(x => x.Id == filterId)
            .FirstAsync();
        if (filter == null)
        {
            return;
        }

        var refs = await SiteRuleUsageInspector.FindCcFilterUsagesAsync(_db, filterId, cancellationToken);
        if (refs.Count > 0)
        {
            throw new InvalidOperationException("cc filter is still referenced by rule groups");
        }

        await _db.Deleteable<CcFilter>()
            .Where(x => x.Id == filterId)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("cc_filter", new[] { resourceId }, cancellationToken);
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
        [JsonPropertyName("resource_type")]
        public string? ResourceType { get; init; }

        [JsonPropertyName("resource_id")]
        public long ResourceId { get; init; }
    }
}
