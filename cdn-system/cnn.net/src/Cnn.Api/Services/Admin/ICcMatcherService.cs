using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface ICcMatcherService
{
    Task<ServiceResult<CcListResult<CcMatcherListItem>>> ListAsync(CcListQuery query, long? userId, bool userScope, CancellationToken cancellationToken);

    Task<ServiceResult<CcMatcherDetailDto>> GetAsync(long id, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> CreateAsync(CcMatcherUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> UpdateAsync(long id, CcMatcherUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isUserRequest, CancellationToken cancellationToken);
}
