using SqlSugar;

namespace Cnn.Api.Services.Deletion;

public sealed class ProductPlanDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public ProductPlanDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.ProductPlan;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "套餐 ID 无效");
        }

        var refs = await _db.Ado.SqlQueryAsync<PlanSubscriptionReference>(
            """
SELECT
  up.id,
  COALESCE(up.record_id, up.name, CAST(up.id AS CHAR)) AS subscription_no,
  u.name AS username
FROM user_package up
JOIN `user` u ON u.id = up.uid
WHERE up.package = @planId
ORDER BY up.id ASC
""",
            new { planId = resourceId });

        if (refs.Count == 0)
        {
            return DeleteGuardResult.Allow();
        }

        var items = refs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.Subscription,
            ResourceId = r.Id,
            DisplayName = $"{r.Username} / {r.SubscriptionNo}",
            Relation = "user_package.package"
        }).ToList();

        return DeleteGuardResult.Deny(
            "PRODUCT_PLAN_IN_USE",
            "套餐仍存在已售套餐记录，必须先移除所有已售套餐后才能删除套餐。",
            items);
    }

    private sealed class PlanSubscriptionReference
    {
        public long Id { get; init; }
        public string SubscriptionNo { get; init; } = string.Empty;
        public string Username { get; init; } = string.Empty;
    }
}
