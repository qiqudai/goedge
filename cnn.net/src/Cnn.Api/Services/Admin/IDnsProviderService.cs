using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IDnsProviderService
{
    Task<ServiceResult<DnsProviderListResult>> ListProvidersAsync(DnsProviderListQuery query, CancellationToken cancellationToken);
    Task<ServiceResult<DnsProviderTypesResult>> GetProviderTypesAsync(CancellationToken cancellationToken);
    Task<ServiceResult<bool>> CreateProviderAsync(DnsProviderCreateRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateProviderAsync(long id, DnsProviderUpdateRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteProviderAsync(long id, CancellationToken cancellationToken);
    Task<ServiceResult<DnsTestResult>> TestAsync(CancellationToken cancellationToken);
    Task<ServiceResult<DnsFixResult>> FixRecordsAsync(CancellationToken cancellationToken);
    Task<ServiceResult<DnsCleanupResult>> CleanupRecordsAsync(CancellationToken cancellationToken);
}
