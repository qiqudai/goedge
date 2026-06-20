using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface ICnameDomainService
{
    Task<ServiceResult<CnameDomainListResult>> ListAsync(CancellationToken cancellationToken);
    Task<ServiceResult<bool>> CreateAsync(CnameDomainUpsertRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, CnameDomainUpsertRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken);
}
