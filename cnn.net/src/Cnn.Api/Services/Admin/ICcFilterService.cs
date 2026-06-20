using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface ICcFilterService
{
    Task<ServiceResult<CcListResult<CcFilterListItem>>> ListAsync(CcListQuery query, long? userId, bool userScope, CancellationToken cancellationToken);

    Task<ServiceResult<CcFilterDetailDto>> GetAsync(long id, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> CreateAsync(CcFilterUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> UpdateAsync(long id, CcFilterUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isUserRequest, CancellationToken cancellationToken);
}
