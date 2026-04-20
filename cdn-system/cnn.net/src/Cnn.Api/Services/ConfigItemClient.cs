using Cnn.Common.Contracts.Admin;

namespace Cnn.Api.Services;

public sealed class ConfigItemClient
{
    private readonly ApiClient _api;

    public ConfigItemClient(ApiClient api)
    {
        _api = api;
    }

    public async Task<IReadOnlyList<ConfigItemDto>> GetAsync(string type, string scopeName = "global", int scopeId = 0)
    {
        var query = new Dictionary<string, string?>
        {
            ["type"] = type,
            ["scope_name"] = scopeName,
            ["scope_id"] = scopeId.ToString()
        };
        var response = await _api.GetAsync<IReadOnlyList<ConfigItemDto>>("config_items", ApiScope.Admin, query);
        return response?.Data ?? Array.Empty<ConfigItemDto>();
    }

    public async Task<bool> SaveAsync(string type, string scopeName, int scopeId, IEnumerable<ConfigItemPayloadDto> items)
    {
        var request = new ConfigItemUpsertRequest
        {
            Type = type,
            ScopeName = scopeName,
            ScopeId = scopeId,
            Items = items.ToList()
        };
        var response = await _api.PostAsync<bool>("config_items", request, ApiScope.Admin);
        return response?.Data ?? false;
    }
}
