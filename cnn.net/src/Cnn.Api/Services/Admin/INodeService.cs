using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface INodeService
{
    Task<ServiceResult<NodeListResult>> ListAsync(NodeListQuery query, CancellationToken cancellationToken);

    Task<ServiceResult<NodeMonitorLogResult>> ListMonitorLogsAsync(long nodeId, NodeMonitorLogQuery query, CancellationToken cancellationToken);

    Task<ServiceResult<NodeListItem>> CreateAsync(NodeCreateRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> UpdateAsync(long nodeId, NodeUpdateRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> UpdateStatusAsync(long nodeId, NodeStatusRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> UpdateAntiBlockingAsync(long nodeId, NodeAntiBlockingRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> DeleteAsync(long nodeId, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> BatchAsync(NodeBatchRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<NodeInstallResult>> InstallAsync(long nodeId, CancellationToken cancellationToken);
}
