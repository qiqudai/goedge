using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class CcRuleGroupDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public CcRuleGroupDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.CcRuleGroup;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "CC 规则组 ID 无效");
        }

        var refs = await SiteRuleUsageInspector.FindCcRuleUsagesAsync(_db, resourceId, cancellationToken);
        if (refs.Count == 0)
        {
            return DeleteGuardResult.Allow();
        }

        var items = refs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.Site,
            ResourceId = r.Id,
            DisplayName = $"{r.Username} / {r.PrimaryDomain}",
            Relation = "site.cc_default_rule"
        }).ToList();

        return DeleteGuardResult.Deny(
            "CC_RULE_GROUP_IN_USE",
            "CC 规则组仍被站点使用，请先解除站点绑定或删除站点后再删除规则组。",
            items);
    }
}
