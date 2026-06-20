using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface INodeGroupService
{
    Task<ServiceResult<NodeGroupListResult>> ListAsync(NodeGroupListQuery query, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> CreateAsync(NodeGroupUpsertRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> UpdateAsync(long groupId, NodeGroupUpsertRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> DeleteAsync(long groupId, CancellationToken cancellationToken);

    Task<ServiceResult<NodeGroupResolutionResult>> GetResolutionAsync(long groupId, NodeGroupResolutionQuery query, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> AssignResolutionAsync(long groupId, NodeGroupAssignRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> ResolutionActionAsync(long groupId, NodeGroupActionRequest request, CancellationToken cancellationToken);
}
