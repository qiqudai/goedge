using System.Text.Json;
using Cnn.Api.Services.Deletion;

namespace Cnn.Api.Services.Tasks.Workflow;

public static class CertificateActionCommandFactory
{
    public static RequestActionCommand CreateStatusChange(
        IReadOnlyCollection<long> certificateIds,
        bool enable,
        bool disableAutoRenew,
        long? ownerUserId = null,
        long? operatorUserId = null)
    {
        var normalizedIds = NormalizeIds(certificateIds);
        if (normalizedIds.Count == 0)
        {
            throw new ArgumentOutOfRangeException(nameof(certificateIds));
        }

        return new RequestActionCommand
        {
            TaskType = enable ? AsyncTaskTypes.CertificateEnable : AsyncTaskTypes.CertificateDisable,
            ResourceType = ResourceTypes.Certificate,
            ResourceId = normalizedIds.Count == 1 ? normalizedIds[0] : null,
            OwnerUserId = ownerUserId,
            OperatorUserId = operatorUserId,
            DedupeKey = $"certificate-status:{(enable ? "enable" : "disable")}:{(disableAutoRenew ? "force" : "normal")}:{string.Join(",", normalizedIds)}",
            PayloadJson = JsonSerializer.Serialize(new
            {
                resource_type = ResourceTypes.Certificate,
                resource_ids = normalizedIds,
                enable,
                disable_auto_renew = disableAutoRenew
            })
        };
    }

    private static List<long> NormalizeIds(IReadOnlyCollection<long> certificateIds)
    {
        return certificateIds
            .Where(id => id > 0)
            .Distinct()
            .OrderBy(id => id)
            .ToList();
    }
}
