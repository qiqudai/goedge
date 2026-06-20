using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class CcMatcherDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public CcMatcherDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.CcMatcher;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "CC 匹配器 ID 无效");
        }

        var refs = await SiteRuleUsageInspector.FindCcMatcherUsagesAsync(_db, resourceId, cancellationToken);
        if (refs.Count == 0)
        {
            return DeleteGuardResult.Allow();
        }

        var items = refs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.CcRuleGroup,
            ResourceId = r.Id,
            DisplayName = string.IsNullOrWhiteSpace(r.Username) ? r.Name : $"{r.Username} / {r.Name}",
            Relation = "cc_rule.rules[].matcher_id"
        }).ToList();

        return DeleteGuardResult.Deny(
            "CC_MATCHER_IN_USE",
            "CC 匹配器仍被规则组引用，请先移除规则组中的关联项后再删除匹配器。",
            items);
    }
}
