using System.Text.Json;
using Cnn.Api.Services.Deletion;

namespace Cnn.Api.Services.Tasks.Workflow;

public static class SiteActionCommandFactory
{
    public static RequestActionCommand CreateDelete(
        IReadOnlyCollection<long> siteIds,
        long? ownerUserId = null,
        long? operatorUserId = null)
    {
        var normalizedIds = NormalizeIds(siteIds);
        if (normalizedIds.Count == 0)
        {
            throw new ArgumentOutOfRangeException(nameof(siteIds));
        }

        return new RequestActionCommand
        {
            TaskType = AsyncTaskTypes.SiteBatchDelete,
            ResourceType = ResourceTypes.Site,
            ResourceId = normalizedIds.Count == 1 ? normalizedIds[0] : null,
            OwnerUserId = ownerUserId,
            OperatorUserId = operatorUserId,
            DedupeKey = $"site-batch-delete:{string.Join(",", normalizedIds)}",
            PayloadJson = JsonSerializer.Serialize(new
            {
                resource_type = ResourceTypes.Site,
                resource_ids = normalizedIds
            })
        };
    }

    public static RequestActionCommand CreateStatusChange(
        IReadOnlyCollection<long> siteIds,
        bool enable,
        long? ownerUserId = null,
        long? operatorUserId = null)
    {
        var normalizedIds = NormalizeIds(siteIds);
        if (normalizedIds.Count == 0)
        {
            throw new ArgumentOutOfRangeException(nameof(siteIds));
        }

        return new RequestActionCommand
        {
            TaskType = enable ? AsyncTaskTypes.SiteEnable : AsyncTaskTypes.SiteDisable,
            ResourceType = ResourceTypes.Site,
            ResourceId = normalizedIds.Count == 1 ? normalizedIds[0] : null,
            OwnerUserId = ownerUserId,
            OperatorUserId = operatorUserId,
            DedupeKey = $"site-status:{(enable ? "enable" : "disable")}:{string.Join(",", normalizedIds)}",
            PayloadJson = JsonSerializer.Serialize(new
            {
                resource_type = ResourceTypes.Site,
                resource_ids = normalizedIds,
                enable
            })
        };
    }

    private static List<long> NormalizeIds(IReadOnlyCollection<long> siteIds)
    {
        return siteIds
            .Where(id => id > 0)
            .Distinct()
            .OrderBy(id => id)
            .ToList();
    }
}
