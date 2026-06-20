using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface ISiteService
{
    Task<ServiceResult<SiteListResult>> ListAsync(SiteListQuery query, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteDetailDto>> GetAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteDetailDto>> CreateAsync(SiteCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteDetailDto>> UpdateAsync(long id, SiteUpdateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteBatchCreateResult>> BatchCreateAsync(SiteBatchCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteBatchProgressResult>> BatchProgressAsync(string batchId, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> BatchUpdateAsync(SiteBatchUpdateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteBatchActionResult>> BatchActionAsync(SiteBatchActionRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteApplyCertResult>> ApplyCertAsync(SiteApplyCertRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteExportResult>> ExportAsync(SiteListQuery query, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<SiteResolveResult>> ResolveAsync(string domain, CancellationToken cancellationToken);
}
