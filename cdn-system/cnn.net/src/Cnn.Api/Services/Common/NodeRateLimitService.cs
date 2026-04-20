using System.Collections.Concurrent;

namespace Cnn.Api.Services.Common;

public interface INodeRateLimitService
{
    void MarkLimited(long nodeId, TimeSpan cooldown);
    bool IsLimited(long nodeId);
}

public sealed class NodeRateLimitService : INodeRateLimitService
{
    private readonly ConcurrentDictionary<long, DateTime> _cooldowns = new();

    public void MarkLimited(long nodeId, TimeSpan cooldown)
    {
        if (nodeId <= 0 || cooldown <= TimeSpan.Zero)
        {
            return;
        }

        _cooldowns[nodeId] = DateTime.UtcNow.Add(cooldown);
    }

    public bool IsLimited(long nodeId)
    {
        if (nodeId <= 0)
        {
            return false;
        }

        if (!_cooldowns.TryGetValue(nodeId, out var until))
        {
            return false;
        }

        if (until <= DateTime.UtcNow)
        {
            _cooldowns.TryRemove(nodeId, out _);
            return false;
        }

        return true;
    }
}
