using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IRegionService
{
    Task<ServiceResult<RegionListResult>> ListAsync(CancellationToken cancellationToken);

    Task<ServiceResult<bool>> CreateAsync(RegionUpsertRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> UpdateAsync(long regionId, RegionUpsertRequest request, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> DeleteAsync(long regionId, CancellationToken cancellationToken);
}
