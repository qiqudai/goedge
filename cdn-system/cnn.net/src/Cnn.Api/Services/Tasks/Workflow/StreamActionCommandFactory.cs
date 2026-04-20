using System.Text.Json;
using Cnn.Api.Services.Deletion;

namespace Cnn.Api.Services.Tasks.Workflow;

public static class StreamActionCommandFactory
{
    public static RequestActionCommand CreateDelete(
        IReadOnlyCollection<long> streamIds,
        long? ownerUserId = null,
        long? operatorUserId = null)
    {
        var normalizedIds = NormalizeIds(streamIds);
        if (normalizedIds.Count == 0)
        {
            throw new ArgumentOutOfRangeException(nameof(streamIds));
        }

        return new RequestActionCommand
        {
            TaskType = AsyncTaskTypes.StreamBatchDelete,
            ResourceType = ResourceTypes.StreamApp,
            ResourceId = normalizedIds.Count == 1 ? normalizedIds[0] : null,
            OwnerUserId = ownerUserId,
            OperatorUserId = operatorUserId,
            DedupeKey = $"stream-batch-delete:{string.Join(",", normalizedIds)}",
            PayloadJson = JsonSerializer.Serialize(new
            {
                resource_type = ResourceTypes.StreamApp,
                resource_ids = normalizedIds
            })
        };
    }

    public static RequestActionCommand CreateStatusChange(
        IReadOnlyCollection<long> streamIds,
        bool enable,
        long? ownerUserId = null,
        long? operatorUserId = null)
    {
        var normalizedIds = NormalizeIds(streamIds);
        if (normalizedIds.Count == 0)
        {
            throw new ArgumentOutOfRangeException(nameof(streamIds));
        }

        return new RequestActionCommand
        {
            TaskType = enable ? AsyncTaskTypes.StreamEnable : AsyncTaskTypes.StreamDisable,
            ResourceType = ResourceTypes.StreamApp,
            ResourceId = normalizedIds.Count == 1 ? normalizedIds[0] : null,
            OwnerUserId = ownerUserId,
            OperatorUserId = operatorUserId,
            DedupeKey = $"stream-status:{(enable ? "enable" : "disable")}:{string.Join(",", normalizedIds)}",
            PayloadJson = JsonSerializer.Serialize(new
            {
                resource_type = ResourceTypes.StreamApp,
                resource_ids = normalizedIds,
                enable
            })
        };
    }

    private static List<long> NormalizeIds(IReadOnlyCollection<long> streamIds)
    {
        return streamIds
            .Where(id => id > 0)
            .Distinct()
            .OrderBy(id => id)
            .ToList();
    }
}
