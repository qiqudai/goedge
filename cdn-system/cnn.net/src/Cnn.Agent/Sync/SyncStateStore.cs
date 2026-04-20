using System.Text.Json;
using Cnn.Agent.Config;

namespace Cnn.Agent.Sync;

public interface ISyncStateStore
{
    SyncStateSnapshot Read();
    void MarkApplied(long version);
    void MarkApplyError(long version, string? error, string? traceId);
}

public sealed record SyncStateSnapshot(
    long LastAppliedVersion,
    long LastErrorVersion,
    string? LastApplyError,
    string? LastErrorTraceId,
    DateTimeOffset UpdatedAt);

public sealed class SyncStateStore : ISyncStateStore
{
    private readonly AgentRuntimePaths _paths;
    private readonly object _lock = new();
    private SyncStateSnapshot _snapshot;

    public SyncStateStore(AgentRuntimePaths paths)
    {
        _paths = paths;
        _snapshot = LoadPersisted();
    }

    public SyncStateSnapshot Read()
    {
        lock (_lock)
        {
            return _snapshot;
        }
    }

    public void MarkApplied(long version)
    {
        if (version <= 0)
        {
            return;
        }

        lock (_lock)
        {
            if (version < _snapshot.LastAppliedVersion)
            {
                return;
            }

            _snapshot = _snapshot with
            {
                LastAppliedVersion = version,
                LastApplyError = null,
                LastErrorTraceId = null,
                LastErrorVersion = 0,
                UpdatedAt = DateTimeOffset.UtcNow
            };

            PersistLocked();
        }
    }

    public void MarkApplyError(long version, string? error, string? traceId)
    {
        lock (_lock)
        {
            _snapshot = _snapshot with
            {
                LastErrorVersion = version,
                LastApplyError = error,
                LastErrorTraceId = traceId,
                UpdatedAt = DateTimeOffset.UtcNow
            };

            PersistLocked();
        }
    }

    private SyncStateSnapshot LoadPersisted()
    {
        try
        {
            if (!File.Exists(_paths.SyncStatePath))
            {
                return new SyncStateSnapshot(0, 0, null, null, DateTimeOffset.UtcNow);
            }

            var json = File.ReadAllText(_paths.SyncStatePath);
            var payload = JsonSerializer.Deserialize<SyncStateFilePayload>(json, new JsonSerializerOptions(JsonSerializerDefaults.Web)
            {
                PropertyNameCaseInsensitive = true
            });

            if (payload == null)
            {
                return new SyncStateSnapshot(0, 0, null, null, DateTimeOffset.UtcNow);
            }

            return new SyncStateSnapshot(
                LastAppliedVersion: payload.LastAppliedVersion,
                LastErrorVersion: payload.LastErrorVersion,
                LastApplyError: payload.LastApplyError,
                LastErrorTraceId: payload.LastErrorTraceId,
                UpdatedAt: payload.UpdatedAt == default ? DateTimeOffset.UtcNow : payload.UpdatedAt);
        }
        catch
        {
            return new SyncStateSnapshot(0, 0, null, null, DateTimeOffset.UtcNow);
        }
    }

    private void PersistLocked()
    {
        try
        {
            Directory.CreateDirectory(_paths.ConfDir);
            var payload = new SyncStateFilePayload
            {
                LastAppliedVersion = _snapshot.LastAppliedVersion,
                LastErrorVersion = _snapshot.LastErrorVersion,
                LastApplyError = _snapshot.LastApplyError,
                LastErrorTraceId = _snapshot.LastErrorTraceId,
                UpdatedAt = _snapshot.UpdatedAt
            };

            var json = JsonSerializer.Serialize(payload, new JsonSerializerOptions(JsonSerializerDefaults.Web)
            {
                WriteIndented = true
            });
            var temp = _paths.SyncStatePath + ".tmp";
            File.WriteAllText(temp, json);
            File.Move(temp, _paths.SyncStatePath, true);
        }
        catch
        {
            // ignore persistence failures
        }
    }

    private sealed class SyncStateFilePayload
    {
        public long LastAppliedVersion { get; set; }
        public long LastErrorVersion { get; set; }
        public string? LastApplyError { get; set; }
        public string? LastErrorTraceId { get; set; }
        public DateTimeOffset UpdatedAt { get; set; }
    }
}
