using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using Cnn.Agent.Config;

namespace Cnn.Agent.Tasks;

public interface ITaskIdempotencyStore
{
    bool TryBegin(long taskId, string taskType, string payloadHash);
    void MarkDone(long taskId, string resultHash);
    bool IsDone(long taskId, out string? resultHash);
    bool IsRunning(long taskId);
    void SaveAck(long taskId, string status, object? applied, string? ret, string? error);
    bool TryGetAck(long taskId, out TaskAckReplay? ack);
}

public sealed record TaskAckReplay(string Status, string? AppliedJson, string? Ret, string? Error);

public sealed class TaskIdempotencyStore : ITaskIdempotencyStore
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly AgentRuntimePaths _paths;
    private readonly object _lock = new();
    private readonly Dictionary<long, TaskRecord> _records = new();
    private const int MaxRecords = 20000;

    public TaskIdempotencyStore(AgentRuntimePaths paths)
    {
        _paths = paths;
        LoadPersisted();
    }

    public bool TryBegin(long taskId, string taskType, string payloadHash)
    {
        if (taskId <= 0)
        {
            return true;
        }

        lock (_lock)
        {
            if (_records.TryGetValue(taskId, out var existing))
            {
                if (existing.State == "done")
                {
                    return false;
                }

                return false;
            }

            _records[taskId] = new TaskRecord
            {
                TaskId = taskId,
                TaskType = taskType?.Trim() ?? string.Empty,
                PayloadHash = payloadHash?.Trim() ?? string.Empty,
                State = "running",
                UpdatedAt = DateTimeOffset.UtcNow
            };
            TrimLocked();
            PersistLocked();
            return true;
        }
    }

    public void MarkDone(long taskId, string resultHash)
    {
        if (taskId <= 0)
        {
            return;
        }

        lock (_lock)
        {
            if (!_records.TryGetValue(taskId, out var record))
            {
                record = new TaskRecord
                {
                    TaskId = taskId,
                    State = "done"
                };
                _records[taskId] = record;
            }

            record.State = "done";
            record.ResultHash = resultHash;
            record.UpdatedAt = DateTimeOffset.UtcNow;
            PersistLocked();
        }
    }

    public bool IsDone(long taskId, out string? resultHash)
    {
        resultHash = null;
        if (taskId <= 0)
        {
            return false;
        }

        lock (_lock)
        {
            if (!_records.TryGetValue(taskId, out var record) || record.State != "done")
            {
                return false;
            }

            resultHash = record.ResultHash;
            return true;
        }
    }

    public bool IsRunning(long taskId)
    {
        if (taskId <= 0)
        {
            return false;
        }

        lock (_lock)
        {
            return _records.TryGetValue(taskId, out var record) && record.State == "running";
        }
    }

    public void SaveAck(long taskId, string status, object? applied, string? ret, string? error)
    {
        if (taskId <= 0)
        {
            return;
        }

        lock (_lock)
        {
            if (!_records.TryGetValue(taskId, out var record))
            {
                record = new TaskRecord { TaskId = taskId };
                _records[taskId] = record;
            }

            record.AckStatus = status?.Trim();
            record.AckAppliedJson = applied == null ? null : JsonSerializer.Serialize(applied, JsonOptions);
            record.AckRet = ret;
            record.AckError = error;
            record.UpdatedAt = DateTimeOffset.UtcNow;

            var hashSource = $"{record.AckStatus}|{record.AckAppliedJson}|{ret}|{error}";
            record.ResultHash = ComputeHash(hashSource);
            record.State = IsTerminal(status) ? "done" : record.State;

            PersistLocked();
        }
    }

    public bool TryGetAck(long taskId, out TaskAckReplay? ack)
    {
        ack = null;
        if (taskId <= 0)
        {
            return false;
        }

        lock (_lock)
        {
            if (!_records.TryGetValue(taskId, out var record))
            {
                return false;
            }

            if (string.IsNullOrWhiteSpace(record.AckStatus))
            {
                return false;
            }

            ack = new TaskAckReplay(record.AckStatus!, record.AckAppliedJson, record.AckRet, record.AckError);
            return true;
        }
    }

    private void LoadPersisted()
    {
        try
        {
            if (!File.Exists(_paths.TaskIdempotencyPath))
            {
                return;
            }

            var json = File.ReadAllText(_paths.TaskIdempotencyPath);
            var payload = JsonSerializer.Deserialize<TaskStorePayload>(json, JsonOptions);
            if (payload?.Records == null || payload.Records.Count == 0)
            {
                return;
            }

            foreach (var item in payload.Records)
            {
                if (item.TaskId <= 0)
                {
                    continue;
                }

                if (item.State == "running")
                {
                    // Restart-safe behavior: running state should not block new execution forever.
                    item.State = "stale";
                }

                _records[item.TaskId] = item;
            }
        }
        catch
        {
            // ignore malformed store
        }
    }

    private void PersistLocked()
    {
        try
        {
            Directory.CreateDirectory(_paths.ConfDir);
            var payload = new TaskStorePayload
            {
                UpdatedAt = DateTimeOffset.UtcNow,
                Records = _records.Values.OrderByDescending(static r => r.UpdatedAt).ToList()
            };
            var json = JsonSerializer.Serialize(payload, new JsonSerializerOptions(JsonSerializerDefaults.Web)
            {
                WriteIndented = true
            });
            var temp = _paths.TaskIdempotencyPath + ".tmp";
            File.WriteAllText(temp, json);
            File.Move(temp, _paths.TaskIdempotencyPath, true);
        }
        catch
        {
            // ignore
        }
    }

    private void TrimLocked()
    {
        if (_records.Count <= MaxRecords)
        {
            return;
        }

        var removeIds = _records.Values
            .OrderBy(static r => r.UpdatedAt)
            .Take(_records.Count - MaxRecords)
            .Select(static r => r.TaskId)
            .ToArray();

        foreach (var id in removeIds)
        {
            _records.Remove(id);
        }
    }

    private static bool IsTerminal(string? status)
    {
        if (string.IsNullOrWhiteSpace(status))
        {
            return false;
        }

        return status.Trim().ToLowerInvariant() is "success" or "fail" or "ignored";
    }

    private static string ComputeHash(string text)
    {
        var hash = SHA256.HashData(Encoding.UTF8.GetBytes(text ?? string.Empty));
        return Convert.ToHexString(hash);
    }

    private sealed class TaskStorePayload
    {
        public DateTimeOffset UpdatedAt { get; set; }
        public List<TaskRecord>? Records { get; set; }
    }

    private sealed class TaskRecord
    {
        public long TaskId { get; set; }
        public string TaskType { get; set; } = string.Empty;
        public string PayloadHash { get; set; } = string.Empty;
        public string State { get; set; } = "running";
        public string? ResultHash { get; set; }
        public string? AckStatus { get; set; }
        public string? AckAppliedJson { get; set; }
        public string? AckRet { get; set; }
        public string? AckError { get; set; }
        public DateTimeOffset UpdatedAt { get; set; } = DateTimeOffset.UtcNow;
    }
}
