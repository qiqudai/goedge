using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IMonitorConfigService
{
    Task<ServiceResult<NodeMonitorConfigDto>> GetAsync(CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(NodeMonitorConfigDto config, CancellationToken cancellationToken);
}
