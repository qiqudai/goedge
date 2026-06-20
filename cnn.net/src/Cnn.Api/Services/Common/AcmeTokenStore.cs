using System.Collections.Concurrent;

namespace Cnn.Api.Services.Common;

public interface IAcmeTokenStore
{
    void Put(string token, string value, TimeSpan ttl);
    bool TryGet(string token, out string? value);
    void Delete(string token);
}

public sealed class AcmeTokenStore : IAcmeTokenStore
{
    private sealed record TokenEntry(string Value, DateTime ExpiresAt);

    private readonly ConcurrentDictionary<string, TokenEntry> _entries = new(StringComparer.Ordinal);

    public void Put(string token, string value, TimeSpan ttl)
    {
        if (string.IsNullOrWhiteSpace(token) || string.IsNullOrWhiteSpace(value))
        {
            return;
        }

        var expiresAt = DateTime.UtcNow.Add(ttl);
        _entries[token] = new TokenEntry(value, expiresAt);
    }

    public bool TryGet(string token, out string? value)
    {
        value = null;
        if (string.IsNullOrWhiteSpace(token))
        {
            return false;
        }

        if (!_entries.TryGetValue(token, out var entry))
        {
            return false;
        }

        if (entry.ExpiresAt <= DateTime.UtcNow)
        {
            _entries.TryRemove(token, out _);
            return false;
        }

        value = entry.Value;
        return true;
    }

    public void Delete(string token)
    {
        if (string.IsNullOrWhiteSpace(token))
        {
            return;
        }

        _entries.TryRemove(token, out _);
    }
}


