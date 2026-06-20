using Yarp.ReverseProxy.Configuration;

namespace Cnn.Agent.Proxy;

public sealed class DynamicProxyConfigProvider : IProxyConfigProvider
{
    private readonly object _updateLock = new();
    private DynamicProxyConfig _current;

    public DynamicProxyConfigProvider()
    {
        _current = new DynamicProxyConfig(ProxySnapshot.CreateFallback());
    }

    public bool IsFallbackMode => CurrentSnapshot.IsFallbackMode;

    public ProxySnapshot CurrentSnapshot => Volatile.Read(ref _current).Snapshot;

    public IProxyConfig GetConfig()
    {
        return Volatile.Read(ref _current);
    }

    public ProxyApplyResult TryUpdate(ProxySnapshot snapshot, bool force = false)
    {
        lock (_updateLock)
        {
            var current = Volatile.Read(ref _current);
            if (!force && snapshot.Version > 0 && current.Snapshot.Version == snapshot.Version)
            {
                return ProxyApplyResult.Skipped(snapshot.Version);
            }

            var next = new DynamicProxyConfig(snapshot);
            Volatile.Write(ref _current, next);
            current.SignalChange();
            return ProxyApplyResult.Ok(snapshot.Version);
        }
    }
}
