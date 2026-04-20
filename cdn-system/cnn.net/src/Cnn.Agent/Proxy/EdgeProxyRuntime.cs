using Cnn.Common.Contracts.Agent;
using Microsoft.Extensions.Logging;

namespace Cnn.Agent.Proxy;

public interface IEdgeProxyRuntime
{
    ProxyApplyResult TryApply(EdgeConfigDto config, bool force = false);
    ProxySnapshot GetCurrent();
    ProxySnapshot? GetLastGood();
}

public sealed class EdgeProxyRuntime : IEdgeProxyRuntime
{
    private readonly DynamicProxyConfigProvider _provider;
    private readonly EdgeConfigToYarpCompiler _compiler;
    private readonly ProxyConfigValidator _validator;
    private readonly ILogger<EdgeProxyRuntime> _logger;
    private readonly SemaphoreSlim _applyLock = new(1, 1);

    private ProxySnapshot? _lastGood;

    public EdgeProxyRuntime(
        DynamicProxyConfigProvider provider,
        EdgeConfigToYarpCompiler compiler,
        ProxyConfigValidator validator,
        ILogger<EdgeProxyRuntime> logger)
    {
        _provider = provider;
        _compiler = compiler;
        _validator = validator;
        _logger = logger;
    }

    public ProxyApplyResult TryApply(EdgeConfigDto config, bool force = false)
    {
        _applyLock.Wait();
        try
        {
            var validated = _validator.Validate(config);
            if (!validated.Success)
            {
                return validated;
            }

            var current = _provider.CurrentSnapshot;
            if (!force && config.Version > 0 && config.Version <= current.Version)
            {
                return ProxyApplyResult.Skipped(config.Version);
            }

            ProxySnapshot next;
            try
            {
                next = _compiler.Compile(config);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "proxy compile failed, version={Version}", config.Version);
                return ProxyApplyResult.Fail(config.Version, ex.Message);
            }

            var updated = _provider.TryUpdate(next, force);
            if (!updated.Success)
            {
                return updated;
            }

            if (updated.Status == "ok")
            {
                Volatile.Write(ref _lastGood, next);
                _logger.LogInformation(
                    "proxy apply success version={Version} routes={Routes} clusters={Clusters}",
                    next.Version,
                    next.Routes.Count,
                    next.Clusters.Count);
            }

            return updated;
        }
        finally
        {
            _applyLock.Release();
        }
    }

    public ProxySnapshot GetCurrent()
    {
        return _provider.CurrentSnapshot;
    }

    public ProxySnapshot? GetLastGood()
    {
        return Volatile.Read(ref _lastGood);
    }
}
