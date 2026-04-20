using Microsoft.JSInterop;

namespace Cnn.Api.Services;

public sealed class LocalStorageService
{
    private readonly IJSRuntime _js;

    public LocalStorageService(IJSRuntime js)
    {
        _js = js;
    }

    public ValueTask<string?> GetItemAsync(string key)
    {
        return _js.InvokeAsync<string?>("cnn.storage.get", key);
    }

    public ValueTask SetItemAsync(string key, string? value)
    {
        return _js.InvokeVoidAsync("cnn.storage.set", key, value ?? string.Empty);
    }

    public ValueTask RemoveItemAsync(string key)
    {
        return _js.InvokeVoidAsync("cnn.storage.remove", key);
    }
}
