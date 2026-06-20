namespace Cnn.Api.Services.Auth;

public sealed class InMemoryLimiter
{
    private sealed class LimitEntry
    {
        public int Count;
        public DateTime WindowStart;
        public DateTime BlockedUntil;
        public DateTime LastSeen;
    }

    private readonly Dictionary<string, LimitEntry> _entries = new(StringComparer.OrdinalIgnoreCase);
    private readonly object _lock = new();
    private readonly int _max;
    private readonly TimeSpan _window;
    private readonly TimeSpan _cooldown;

    public InMemoryLimiter(int max, TimeSpan window, TimeSpan cooldown)
    {
        _max = max <= 0 ? 5 : max;
        _window = window <= TimeSpan.Zero ? TimeSpan.FromMinutes(5) : window;
        _cooldown = cooldown <= TimeSpan.Zero ? TimeSpan.FromMinutes(10) : cooldown;
    }

    public (bool Allowed, TimeSpan Cooldown) Allow(string? key)
    {
        var now = DateTime.UtcNow;
        key = string.IsNullOrWhiteSpace(key) ? "unknown" : key.Trim();
        lock (_lock)
        {
            if (!_entries.TryGetValue(key, out var entry))
            {
                entry = new LimitEntry
                {
                    WindowStart = now,
                    LastSeen = now
                };
                _entries[key] = entry;
            }

            entry.LastSeen = now;
            if (now < entry.BlockedUntil)
            {
                return (false, entry.BlockedUntil - now);
            }

            if (now - entry.WindowStart > _window)
            {
                entry.WindowStart = now;
                entry.Count = 0;
            }

            entry.Count++;
            if (entry.Count > _max)
            {
                entry.BlockedUntil = now.Add(_cooldown);
                return (false, _cooldown);
            }
        }

        return (true, TimeSpan.Zero);
    }
}
