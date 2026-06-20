using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface ISiteDefaultService
{
    Task<ServiceResult<SiteDefaultListResult>> ListAsync(SiteDefaultListQuery query, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> CreateAsync(SiteDefaultCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(string name, SiteDefaultUpdateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(string name, string? scopeName, long? scopeId, long? userId, bool isAdmin, CancellationToken cancellationToken);
}
