using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface ISiteGroupService
{
    Task<ServiceResult<SiteGroupListResult>> ListAsync(SiteGroupListQuery query, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteGroupDto>> CreateAsync(SiteGroupUpsertRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, SiteGroupUpsertRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken);
}
