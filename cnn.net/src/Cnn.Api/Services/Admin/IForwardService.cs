using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IForwardService
{
    Task<ServiceResult<ForwardListResult>> ListAsync(ForwardListQuery query, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<ForwardDetailDto>> CreateAsync(ForwardCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<ForwardDetailDto>> UpdateAsync(long id, ForwardUpdateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<ForwardBatchCreateResult>> BatchCreateAsync(ForwardBatchCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> BatchUpdateAsync(ForwardBatchUpdateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<ForwardBatchActionResult>> BatchActionAsync(ForwardBatchActionRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
}
