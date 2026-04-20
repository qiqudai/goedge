using Cnn.Domain.Entities;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface IDnsSyncService
{
    Task<bool> SyncUserDnsRecordsAsync(Site? oldSite, Site? newSite);

    Task<bool> SyncLineRecordsAsync(long groupId, string lineId, string lineName, string action, IReadOnlyList<long> nodeIds);

    Task<bool> SyncPackageCnameForLineChangeAsync(long groupId, string lineId, string lineName, IReadOnlyList<long> nodeIds, string action);

    Task<bool> SyncPackageCnameForNodesAsync(IReadOnlyList<long> nodeIds, string action);

    Task<bool> SyncPackageLineRecordsAsync(CnameDomains domain, string host, long groupId, string lineId, string lineName, string action, IReadOnlyList<long> nodeIds);
}
