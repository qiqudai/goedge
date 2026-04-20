using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IAclService
{
    Task<ServiceResult<AclListResult>> ListAsync(AclListQuery query, CancellationToken cancellationToken);
    Task<ServiceResult<AclDetailDto>> GetAsync(long id, CancellationToken cancellationToken);
    Task<ServiceResult<AclDetailDto>> CreateAsync(AclUpsertRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, AclUpsertRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken);
}
