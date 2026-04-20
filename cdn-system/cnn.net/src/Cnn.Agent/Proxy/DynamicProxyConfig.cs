using Microsoft.Extensions.Primitives;
using Yarp.ReverseProxy.Configuration;

namespace Cnn.Agent.Proxy;

internal sealed class DynamicProxyConfig : IProxyConfig
{
    private readonly CancellationTokenSource _cts;

    public DynamicProxyConfig(ProxySnapshot snapshot)
    {
        Snapshot = snapshot;
        _cts = new CancellationTokenSource();
    }

    public ProxySnapshot Snapshot { get; }

    public IReadOnlyList<RouteConfig> Routes => Snapshot.Routes;

    public IReadOnlyList<ClusterConfig> Clusters => Snapshot.Clusters;

    public IChangeToken ChangeToken => new CancellationChangeToken(_cts.Token);

    public void SignalChange()
    {
        try
        {
            _cts.Cancel();
        }
        catch
        {
            // ignored
        }
        finally
        {
            _cts.Dispose();
        }
    }
}
