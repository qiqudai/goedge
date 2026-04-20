using System.Text.Json;
using Cnn.Agent.Config;

namespace Cnn.Agent.Diagnostics;

public static class DebugSwitchKeys
{
    public const string ShipAccessLogs = "ship_access_logs";
    public const string ShipStreamLogs = "ship_stream_logs";
    public const string ShipSecurityLogs = "ship_security_logs";
    public const string ShipSystemLogs = "ship_system_logs";
    public const string ShipMetrics = "ship_metrics";
    public const string ManualDebugLog = "manual_debug_log";
    public const string RuntimeVerbose = "runtime_verbose";
}

public sealed record DebugSwitchApplyResult(
    int AppliedCount,
    IReadOnlyDictionary<string, bool> Updated,
    IReadOnlyDictionary<string, bool> Current,
    DateTimeOffset? ExpiresAt);

public interface IDebugSwitchStore
{
    bool IsEnabled(string key);
    IReadOnlyDictionary<string, bool> GetSnapshot();
    DebugSwitchApplyResult Apply(IEnumerable<KeyValuePair<string, bool>> updates, string? actor, string? reason, int? ttlSeconds = null);
}

public sealed class DebugSwitchStore : IDebugSwitchStore
{
    private static readonly Dictionary<string, bool> DefaultSwitches = new(StringComparer.OrdinalIgnoreCase)
    {
        [DebugSwitchKeys.ShipAccessLogs] = true,
        [DebugSwitchKeys.ShipStreamLogs] = true,
        [DebugSwitchKeys.ShipSecurityLogs] = true,
        [DebugSwitchKeys.ShipSystemLogs] = true,
        [DebugSwitchKeys.ShipMetrics] = true,
        [DebugSwitchKeys.ManualDebugLog] = true,
        [DebugSwitchKeys.RuntimeVerbose] = false
    };

    private readonly AgentRuntimePaths _paths;
    private readonly object _lock = new();
    private Dictionary<string, bool> _switches;
    private DateTimeOffset? _expiresAt;

    public DebugSwitchStore(AgentRuntimePaths paths)
    {
        _paths = paths;
        _switches = new Dictionary<string, bool>(DefaultSwitches, StringComparer.OrdinalIgnoreCase);
        LoadPersisted();
    }

    public bool IsEnabled(string key)
    {
        if (string.IsNullOrWhiteSpace(key))
        {
            return false;
        }

        var normalized = NormalizeKey(key);
        lock (_lock)
        {
            EnsureNotExpiredLocked();
            return _switches.TryGetValue(normalized, out var enabled) && enabled;
        }
    }

    public IReadOnlyDictionary<string, bool> GetSnapshot()
    {
        lock (_lock)
        {
            EnsureNotExpiredLocked();
            return new Dictionary<string, bool>(_switches, StringComparer.OrdinalIgnoreCase);
        }
    }

    public DebugSwitchApplyResult Apply(IEnumerable<KeyValuePair<string, bool>> updates, string? actor, string? reason, int? ttlSeconds = null)
    {
        if (updates == null)
        {
            return new DebugSwitchApplyResult(0, new Dictionary<string, bool>(), GetSnapshot(), _expiresAt);
        }

        var changed = new Dictionary<string, bool>(StringComparer.OrdinalIgnoreCase);
        var metaChanged = false;
        lock (_lock)
        {
            EnsureNotExpiredLocked();

            foreach (var entry in updates)
            {
                var normalized = NormalizeKey(entry.Key);
                if (string.IsNullOrWhiteSpace(normalized))
                {
                    continue;
                }

                if (_switches.TryGetValue(normalized, out var existing) && existing == entry.Value)
                {
                    continue;
                }

                _switches[normalized] = entry.Value;
                changed[normalized] = entry.Value;
            }

            if (ttlSeconds.HasValue)
            {
                if (ttlSeconds.Value > 0)
                {
                    _expiresAt = DateTimeOffset.UtcNow.AddSeconds(ttlSeconds.Value);
                }
                else
                {
                    _expiresAt = null;
                }
                metaChanged = true;
            }

            if (changed.Count > 0 || metaChanged)
            {
                Persist(actor, reason);
            }

            return new DebugSwitchApplyResult(
                changed.Count,
                new Dictionary<string, bool>(changed, StringComparer.OrdinalIgnoreCase),
                new Dictionary<string, bool>(_switches, StringComparer.OrdinalIgnoreCase),
                _expiresAt);
        }
    }

    private void LoadPersisted()
    {
        try
        {
            if (!File.Exists(_paths.DebugSwitchPath))
            {
                return;
            }

            var json = File.ReadAllText(_paths.DebugSwitchPath);
            var payload = JsonSerializer.Deserialize<DebugSwitchFilePayload>(json);
            if (payload?.Switches == null || payload.Switches.Count == 0)
            {
                return;
            }

            foreach (var (key, value) in payload.Switches)
            {
                var normalized = NormalizeKey(key);
                if (!string.IsNullOrWhiteSpace(normalized))
                {
                    _switches[normalized] = value;
                }
            }

            _expiresAt = payload.ExpiresAt;
        }
        catch
        {
            // ignore malformed persisted switches and continue with defaults
        }
    }

    private void Persist(string? actor, string? reason)
    {
        Directory.CreateDirectory(_paths.ConfDir);

        var payload = new DebugSwitchFilePayload
        {
            UpdatedAt = DateTimeOffset.UtcNow,
            Actor = actor,
            Reason = reason,
            ExpiresAt = _expiresAt,
            Switches = new Dictionary<string, bool>(_switches, StringComparer.OrdinalIgnoreCase)
        };

        var json = JsonSerializer.Serialize(payload, new JsonSerializerOptions(JsonSerializerDefaults.Web)
        {
            WriteIndented = true
        });

        var tempPath = _paths.DebugSwitchPath + ".tmp";
        File.WriteAllText(tempPath, json);
        File.Move(tempPath, _paths.DebugSwitchPath, true);
    }

    private static string NormalizeKey(string key)
    {
        return key.Trim().Replace('-', '_').ToLowerInvariant();
    }

    private void EnsureNotExpiredLocked()
    {
        if (!_expiresAt.HasValue)
        {
            return;
        }

        if (_expiresAt.Value > DateTimeOffset.UtcNow)
        {
            return;
        }

        _switches = new Dictionary<string, bool>(DefaultSwitches, StringComparer.OrdinalIgnoreCase);
        _expiresAt = null;
        Persist("system", "ttl_expired");
    }

    private sealed class DebugSwitchFilePayload
    {
        public DateTimeOffset UpdatedAt { get; set; }
        public string? Actor { get; set; }
        public string? Reason { get; set; }
        public DateTimeOffset? ExpiresAt { get; set; }
        public Dictionary<string, bool>? Switches { get; set; }
    }
}
