using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class SecurityRuleDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public SecurityRuleDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.SecurityRule;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "规则 ID 无效");
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
            "SECURITY_RULE_IN_USE",
            "规则仍被站点使用，请先解除站点绑定或删除站点后再删除规则。",
            items);
    }
}
