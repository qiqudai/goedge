using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IForwardGroupService
{
    Task<ServiceResult<ForwardGroupListResult>> ListAsync(string? keyword, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<ForwardGroupDto>> CreateAsync(ForwardGroupUpsertRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(ForwardGroupUpsertRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken);
}
