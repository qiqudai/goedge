using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IGlobalConfigService
{
    Task<ServiceResult<GlobalConfigDto>> GetAsync(CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(GlobalConfigDto config, CancellationToken cancellationToken);
}
