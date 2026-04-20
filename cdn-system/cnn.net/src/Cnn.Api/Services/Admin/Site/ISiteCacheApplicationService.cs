using Cnn.Common.Contracts;
using Cnn.Api.Cache;

namespace Cnn.Api.Services.Admin;

public sealed class SiteCacheViewDto
{
    public string? Raw { get; init; }
    public CacheConfigDto? Config { get; init; }
}

public sealed class SiteCacheSaveResultDto
{
    public string? Raw { get; init; }
    public CacheSiteConfigDto? Compiled { get; init; }
}

public interface ISiteCacheApplicationService
{
    Task<ServiceResult<SiteCacheViewDto>> GetAsync(
        int siteId,
        CancellationToken cancellationToken);

    Task<ServiceResult<SiteCacheSaveResultDto>> SaveAsync(
        int siteId,
        CacheConfigDto input,
        bool compile,
        CancellationToken cancellationToken);

    Task<ServiceResult<CacheSiteConfigDto>> CompileAsync(
        int siteId,
        CancellationToken cancellationToken);
}
