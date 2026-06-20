using System.Collections.Concurrent;

namespace Cnn.Api.Services.Common;

public interface INodeStatusService
{
    void MarkOnline(long nodeId, DateTime? at = null);
    void MarkOffline(long nodeId);
    bool IsOnline(long nodeId, TimeSpan ttl);
}

public sealed class NodeStatusService : INodeStatusService
{
    private readonly ConcurrentDictionary<long, DateTime> _lastSeen = new();
    private readonly ConcurrentDictionary<long, bool> _offline = new();

    public void MarkOnline(long nodeId, DateTime? at = null)
    {
        if (nodeId <= 0)
        {
            return;
        }

        var timestamp = at ?? DateTime.UtcNow;
        _lastSeen[nodeId] = timestamp;
        _offline[nodeId] = false;
    }

    public void MarkOffline(long nodeId)
    {
        if (nodeId <= 0)
        {
            return;
        }

        _offline[nodeId] = true;
    }

    public bool IsOnline(long nodeId, TimeSpan ttl)
    {
        if (nodeId <= 0)
        {
            return false;
        }

        if (_offline.TryGetValue(nodeId, out var offline) && offline)
        {
            return false;
        }

        if (!_lastSeen.TryGetValue(nodeId, out var last))
        {
            return false;
        }

        return DateTime.UtcNow - last <= ttl;
    }
}
