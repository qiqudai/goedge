using Cnn.Api.Services.Deletion;

namespace Cnn.Api.Services.Tasks.Workflow;

public static class DeleteRequestCommandFactory
{
    public static RequestDeleteCommand Create(
        string resourceType,
        long resourceId,
        long? ownerUserId = null,
        long? operatorUserId = null,
        string? requestedReason = null)
    {
        if (string.IsNullOrWhiteSpace(resourceType))
        {
            throw new ArgumentException("resourceType is required.", nameof(resourceType));
        }

        if (resourceId <= 0)
        {
            throw new ArgumentOutOfRangeException(nameof(resourceId));
        }

        return new RequestDeleteCommand
        {
            ResourceType = resourceType,
            ResourceId = resourceId,
            OwnerUserId = ownerUserId,
            OperatorUserId = operatorUserId,
            RequestedReason = requestedReason,
            DedupeKey = $"{ResolveKeyPrefix(resourceType)}:{resourceId}"
        };
    }

    private static string ResolveKeyPrefix(string resourceType)
    {
        return resourceType.ToLowerInvariant() switch
        {
            ResourceTypes.Node => "node-delete",
            ResourceTypes.LineGroup => "line-group-delete",
            ResourceTypes.Certificate => "certificate-delete",
            ResourceTypes.SecurityRule => "security-rule-delete",
            ResourceTypes.CcRuleGroup => "cc-rule-group-delete",
            ResourceTypes.CcMatcher => "cc-matcher-delete",
            ResourceTypes.CcFilter => "cc-filter-delete",
            ResourceTypes.AclRule => "acl-rule-delete",
            ResourceTypes.ProductPlan => "product-plan-delete",
            ResourceTypes.Subscription => "subscription-delete",
            ResourceTypes.SiteGroup => "site-group-delete",
            ResourceTypes.Site => "site-delete",
            ResourceTypes.StreamApp => "stream-delete",
            ResourceTypes.StreamGroup => "stream-group-delete",
            ResourceTypes.UserAccount => "user-purge",
            _ => $"{resourceType}-delete"
        };
    }
}
