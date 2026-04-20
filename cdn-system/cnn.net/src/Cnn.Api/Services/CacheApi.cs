using System.Net.Http.Json;
using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts;

namespace Cnn.Api.Services;

public sealed class CacheApi
{
    private readonly HttpClient _http;
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    public CacheApi(HttpClient http)
    {
        _http = http;
    }

    public async Task<CacheGetResult> GetAsync(int siteId)
    {
        var response = await _http.GetFromJsonAsync<ApiResponse<CacheGetData>>($"/api/sites/{siteId}/cache", JsonOptions);
        if (response?.Data == null)
        {
            return new CacheGetResult(null, null, response?.Message);
        }

        return new CacheGetResult(response.Data.Raw, response.Data.Config, response.Message);
    }

    public async Task<CacheSaveResult> SaveAsync(int siteId, CacheConfigDto config, bool compile)
    {
        var response = await _http.PostAsJsonAsync($"/api/sites/{siteId}/cache?compile={compile.ToString().ToLowerInvariant()}", config, JsonOptions);
        var payload = await response.Content.ReadFromJsonAsync<ApiResponse<CacheSaveData>>(JsonOptions);
        if (payload?.Data == null)
        {
            return new CacheSaveResult(null, null, payload?.Message);
        }

        return new CacheSaveResult(payload.Data.Raw, payload.Data.Compiled, payload.Message);
    }

    public async Task<CacheSiteConfigDto?> CompileAsync(int siteId)
    {
        var response = await _http.PostAsync($"/api/sites/{siteId}/cache/compile", null);
        var payload = await response.Content.ReadFromJsonAsync<ApiResponse<CacheSiteConfigDto>>(JsonOptions);
        return payload?.Data;
    }

    private sealed class CacheGetData
    {
        [JsonPropertyName("raw")]
        public string? Raw { get; set; }

        [JsonPropertyName("config")]
        public CacheConfigDto? Config { get; set; }
    }

    private sealed class CacheSaveData
    {
        [JsonPropertyName("raw")]
        public string? Raw { get; set; }

        [JsonPropertyName("compiled")]
        public CacheSiteConfigDto? Compiled { get; set; }
    }
}

public sealed record CacheGetResult(string? Raw, CacheConfigDto? Config, string? Message);

public sealed record CacheSaveResult(string? Raw, CacheSiteConfigDto? Compiled, string? Message);
