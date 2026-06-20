using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Text;
using System.Text.Json;
using Cnn.Common.Contracts;
using Microsoft.AspNetCore.Components;

namespace Cnn.Api.Services;

public enum ApiScope
{
    Admin,
    User,
    Public
}

public sealed class ApiClient
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly HttpClient _http;
    private readonly ClientSession _session;
    private readonly LocalStorageService _storage;
    private readonly NavigationManager _navigation;

    public ApiClient(HttpClient http, ClientSession session, LocalStorageService storage, NavigationManager navigation)
    {
        _http = http;
        _session = session;
        _storage = storage;
        _navigation = navigation;
    }

    public async Task<ApiResponse<T>?> GetAsync<T>(string path, ApiScope? scope = null, Dictionary<string, string?>? query = null)
    {
        await EnsureSessionAsync(scope);
        var url = BuildUrl(path, scope, query);
        using var request = new HttpRequestMessage(HttpMethod.Get, url);
        AttachAuth(request, scope);
        using var response = await _http.SendAsync(request);
        await RefreshTokenAsync(response);
        return await ReadApiResponseAsync<T>(response);
    }

    public async Task<ApiResponse<T>?> PostAsync<T>(string path, object? body, ApiScope? scope = null)
    {
        await EnsureSessionAsync(scope);
        var url = BuildUrl(path, scope, null);
        using var request = new HttpRequestMessage(HttpMethod.Post, url);
        AttachAuth(request, scope);
        if (body != null)
        {
            request.Content = JsonContent.Create(body, options: JsonOptions);
        }
        using var response = await _http.SendAsync(request);
        await RefreshTokenAsync(response);
        return await ReadApiResponseAsync<T>(response);
    }

    public async Task<ApiResponse<T>?> PutAsync<T>(string path, object? body, ApiScope? scope = null)
    {
        await EnsureSessionAsync(scope);
        var url = BuildUrl(path, scope, null);
        using var request = new HttpRequestMessage(HttpMethod.Put, url);
        AttachAuth(request, scope);
        if (body != null)
        {
            request.Content = JsonContent.Create(body, options: JsonOptions);
        }
        using var response = await _http.SendAsync(request);
        await RefreshTokenAsync(response);
        return await ReadApiResponseAsync<T>(response);
    }

    public async Task<ApiResponse<T>?> DeleteAsync<T>(string path, object? body = null, ApiScope? scope = null)
    {
        await EnsureSessionAsync(scope);
        var url = BuildUrl(path, scope, null);
        using var request = new HttpRequestMessage(HttpMethod.Delete, url);
        AttachAuth(request, scope);
        if (body != null)
        {
            request.Content = JsonContent.Create(body, options: JsonOptions);
        }
        using var response = await _http.SendAsync(request);
        await RefreshTokenAsync(response);
        return await ReadApiResponseAsync<T>(response);
    }

    public async Task<ApiResponse<T>?> PostFormAsync<T>(string path, MultipartFormDataContent content, ApiScope? scope = null)
    {
        await EnsureSessionAsync(scope);
        var url = BuildUrl(path, scope, null);
        using var request = new HttpRequestMessage(HttpMethod.Post, url)
        {
            Content = content
        };
        AttachAuth(request, scope);
        using var response = await _http.SendAsync(request);
        await RefreshTokenAsync(response);
        return await ReadApiResponseAsync<T>(response);
    }

    private static string BuildUrl(string path, ApiScope? scope, Dictionary<string, string?>? query)
    {
        var prefix = scope switch
        {
            ApiScope.Public => "/api/v1",
            ApiScope.User => "/api/v1/user",
            _ => "/api/v1/admin"
        };

        var normalized = string.IsNullOrWhiteSpace(path) ? string.Empty : path.Trim();
        if (!normalized.StartsWith("/", StringComparison.Ordinal))
        {
            normalized = "/" + normalized;
        }

        var url = prefix + normalized;
        if (query == null || query.Count == 0)
        {
            return url;
        }

        var builder = new StringBuilder(url);
        var first = true;
        foreach (var pair in query)
        {
            if (string.IsNullOrWhiteSpace(pair.Key))
            {
                continue;
            }

            builder.Append(first ? '?' : '&');
            first = false;
            builder.Append(Uri.EscapeDataString(pair.Key));
            builder.Append('=');
            builder.Append(Uri.EscapeDataString(pair.Value ?? string.Empty));
        }

        return builder.ToString();
    }

    private void AttachAuth(HttpRequestMessage request, ApiScope? scope)
    {
        if (scope == ApiScope.Public)
        {
            return;
        }

        if (!string.IsNullOrWhiteSpace(_session.Token))
        {
            request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", _session.Token);
        }
    }

    private async Task RefreshTokenAsync(HttpResponseMessage response)
    {
        if (response.Headers.TryGetValues("X-Auth-Token", out var values))
        {
            var token = values.FirstOrDefault();
            if (!string.IsNullOrWhiteSpace(token))
            {
                _session.Set(token, _session.Role, _session.Username);
                try
                {
                    await _storage.SetItemAsync("admin_token", token);
                }
                catch
                {
                    // Ignore JS runtime availability errors during prerender/teardown.
                }
            }
        }
    }

    private async Task EnsureSessionAsync(ApiScope? scope)
    {
        if (scope == ApiScope.Public || !string.IsNullOrWhiteSpace(_session.Token))
        {
            return;
        }

        try
        {
            var token = await _storage.GetItemAsync("admin_token");
            if (string.IsNullOrWhiteSpace(token))
            {
                return;
            }

            var role = await _storage.GetItemAsync("role");
            var username = await _storage.GetItemAsync("username");
            _session.Set(token, role, username);
            _session.MarkInitialized();
        }
        catch
        {
            // Ignore JS runtime availability errors during prerender.
        }
    }

    private async Task<ApiResponse<T>?> ReadApiResponseAsync<T>(HttpResponseMessage response)
    {
        if (response.Content == null)
        {
            return null;
        }

        try
        {
            var raw = await response.Content.ReadAsStringAsync();
            if (string.IsNullOrWhiteSpace(raw))
            {
                return null;
            }

            if (await TryHandleMaintenanceAsync(response, raw))
            {
                return null;
            }

            return JsonSerializer.Deserialize<ApiResponse<T>>(raw, JsonOptions);
        }
        catch
        {
            return null;
        }
    }

    private async Task<bool> TryHandleMaintenanceAsync(HttpResponseMessage response, string raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        var path = response.RequestMessage?.RequestUri?.AbsolutePath ?? string.Empty;
        if (!path.StartsWith("/api/v1/user", StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            var root = doc.RootElement;
            if (!root.TryGetProperty("maintenance", out var flag) || flag.ValueKind != JsonValueKind.True)
            {
                if (response.StatusCode != System.Net.HttpStatusCode.ServiceUnavailable)
                {
                    return false;
                }

                if (!root.TryGetProperty("code", out var codeEl) || codeEl.GetInt32() != 503)
                {
                    return false;
                }
            }

            var message = ResolveMessage(root);
            if (string.IsNullOrWhiteSpace(message))
            {
                message = "系统维护中，请稍后再试";
            }

            await _storage.SetItemAsync("maintenance_msg", message);
            if (!_navigation.Uri.EndsWith("/maintenance", StringComparison.OrdinalIgnoreCase))
            {
                _navigation.NavigateTo("/maintenance", forceLoad: true);
            }

            return true;
        }
        catch
        {
            return false;
        }
    }

    private static string ResolveMessage(JsonElement root)
    {
        if (root.TryGetProperty("message", out var msgEl) && msgEl.ValueKind == JsonValueKind.String)
        {
            return msgEl.GetString() ?? string.Empty;
        }

        if (root.TryGetProperty("msg", out var legacyEl) && legacyEl.ValueKind == JsonValueKind.String)
        {
            return legacyEl.GetString() ?? string.Empty;
        }

        if (root.TryGetProperty("data", out var dataEl) && dataEl.ValueKind == JsonValueKind.Object)
        {
            if (dataEl.TryGetProperty("message", out var dataMsg) && dataMsg.ValueKind == JsonValueKind.String)
            {
                return dataMsg.GetString() ?? string.Empty;
            }
        }

        if (root.TryGetProperty("error", out var errEl) && errEl.ValueKind == JsonValueKind.String)
        {
            return errEl.GetString() ?? string.Empty;
        }

        return string.Empty;
    }
}
