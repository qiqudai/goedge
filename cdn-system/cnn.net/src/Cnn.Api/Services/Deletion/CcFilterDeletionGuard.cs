using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class CcFilterDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public CcFilterDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.CcFilter;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "CC 过滤器 ID 无效");
        }

        var refs = await SiteRuleUsageInspector.FindCcFilterUsagesAsync(_db, resourceId, cancellationToken);
        if (refs.Count == 0)
        {
            return DeleteGuardResult.Allow();
        }

        var items = refs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.CcRuleGroup,
            ResourceId = r.Id,
            DisplayName = string.IsNullOrWhiteSpace(r.Username) ? r.Name : $"{r.Username} / {r.Name}",
            Relation = "cc_rule.rules[].filter*_id"
        }).ToList();

        return DeleteGuardResult.Deny(
            "CC_FILTER_IN_USE",
            "CC 过滤器仍被规则组引用，请先移除规则组中的关联项后再删除过滤器。",
            items);
    }
}
