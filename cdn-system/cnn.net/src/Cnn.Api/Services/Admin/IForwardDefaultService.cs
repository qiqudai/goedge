using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IForwardDefaultService
{
    Task<ServiceResult<ForwardDefaultListResult>> ListAsync(long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> CreateAsync(ForwardDefaultCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(ForwardDefaultDeleteRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
}
