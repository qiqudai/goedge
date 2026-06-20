using Cnn.Common.Contracts;

namespace Cnn.Api.Services;

public sealed class SystemInfoClient
{
    private readonly ApiClient _api;

    public SystemInfoClient(ApiClient api)
    {
        _api = api;
    }

    public async Task<SystemInfoDto> GetAsync()
    {
        var response = await _api.GetAsync<SystemInfoDto>("system_info", ApiScope.Public);
        return response?.Data ?? new SystemInfoDto();
    }
}
