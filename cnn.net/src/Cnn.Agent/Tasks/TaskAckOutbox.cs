using System.Text.Json;
using Cnn.Agent.Config;

namespace Cnn.Agent.Tasks;

public sealed record TaskOutboxItem(
    long Id,
    string Kind,
    long? TaskId,
    string Payload,
    int Attempts,
    string? LastError,
    DateTimeOffset CreatedAt,
    DateTimeOffset? LastAttemptAt);

public interface ITaskAckOutbox
{
    long Enqueue(string kind, long? taskId, string payload);
    IReadOnlyList<TaskOutboxItem> ListPending(int limit);
    void MarkSent(long id);
    void MarkFailed(long id, string? error);
}

public sealed class TaskAckOutbox : ITaskAckOutbox
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly AgentRuntimePaths _paths;
    private readonly object _lock = new();
    private readonly Dictionary<long, TaskOutboxItem> _items = new();
    private long _nextId = 1;
    private const int MaxPending = 5000;

    public TaskAckOutbox(AgentRuntimePaths paths)
    {
        _paths = paths;
        LoadPersisted();
    }

    public long Enqueue(string kind, long? taskId, string payload)
    {
        if (string.IsNullOrWhiteSpace(payload))
        {
            return 0;
        }

        lock (_lock)
        {
            var id = _nextId++;
            _items[id] = new TaskOutboxItem(
                Id: id,
                Kind: string.IsNullOrWhiteSpace(kind) ? "task_ack" : kind.Trim(),
                TaskId: taskId,
                Payload: payload,
                Attempts: 0,
                LastError: null,
                CreatedAt: DateTimeOffset.UtcNow,
                LastAttemptAt: null);

            TrimLocked();
            PersistLocked();
            return id;
        }
    }

    public IReadOnlyList<TaskOutboxItem> ListPending(int limit)
    {
        if (limit <= 0)
        {
            limit = 20;
        }

        lock (_lock)
        {
            return _items.Values
                .OrderBy(static i => i.CreatedAt)
                .Take(limit)
                .ToArray();
        }
    }

    public void MarkSent(long id)
    {
        if (id <= 0)
        {
            return;
        }

        lock (_lock)
        {
            if (_items.Remove(id))
            {
                PersistLocked();
            }
        }
    }

    public void MarkFailed(long id, string? error)
    {
        if (id <= 0)
        {
            return;
        }

        lock (_lock)
        {
            if (!_items.TryGetValue(id, out var item))
            {
                return;
            }

            _items[id] = item with
            {
                Attempts = item.Attempts + 1,
                LastError = string.IsNullOrWhiteSpace(error) ? "send_failed" : error,
                LastAttemptAt = DateTimeOffset.UtcNow
            };
            PersistLocked();
        }
    }

    private void LoadPersisted()
    {
        try
        {
            if (!File.Exists(_paths.TaskAckOutboxPath))
            {
                return;
            }

            var json = File.ReadAllText(_paths.TaskAckOutboxPath);
            var payload = JsonSerializer.Deserialize<OutboxPayload>(json, JsonOptions);
            if (payload == null)
            {
                return;
            }

            _nextId = payload.NextId > 0 ? payload.NextId : 1;
            if (payload.Items == null || payload.Items.Count == 0)
            {
                return;
            }

            foreach (var item in payload.Items)
            {
                if (item.Id <= 0 || string.IsNullOrWhiteSpace(item.Payload))
                {
                    continue;
                }

                _items[item.Id] = item;
                if (item.Id >= _nextId)
                {
                    _nextId = item.Id + 1;
                }
            }
        }
        catch
        {
            // ignore corrupted outbox
        }
    }

    private void PersistLocked()
    {
        try
        {
            Directory.CreateDirectory(_paths.ConfDir);
            var payload = new OutboxPayload
            {
                UpdatedAt = DateTimeOffset.UtcNow,
                NextId = _nextId,
                Items = _items.Values.OrderBy(static i => i.CreatedAt).ToList()
            };

            var json = JsonSerializer.Serialize(payload, new JsonSerializerOptions(JsonSerializerDefaults.Web)
            {
                WriteIndented = true
            });
            var temp = _paths.TaskAckOutboxPath + ".tmp";
            File.WriteAllText(temp, json);
            File.Move(temp, _paths.TaskAckOutboxPath, true);
        }
        catch
        {
            // ignore
        }
    }

    private void TrimLocked()
    {
        if (_items.Count <= MaxPending)
        {
            return;
        }

        var toRemove = _items.Values
            .OrderBy(static i => i.CreatedAt)
            .Take(_items.Count - MaxPending)
            .Select(static i => i.Id)
            .ToArray();

        foreach (var id in toRemove)
        {
            _items.Remove(id);
        }
    }

    private sealed class OutboxPayload
    {
        public DateTimeOffset UpdatedAt { get; set; }
        public long NextId { get; set; }
        public List<TaskOutboxItem>? Items { get; set; }
    }
}
