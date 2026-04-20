using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.OutputCaching;
using Microsoft.Extensions.Logging;

namespace Cnn.Agent.Cache;

public sealed class DynamicCachePolicy : IOutputCachePolicy
{
    private readonly CacheRuntimeStore _store;
    private readonly ILogger<DynamicCachePolicy> _logger;

    public DynamicCachePolicy(CacheRuntimeStore store, ILogger<DynamicCachePolicy> logger)
    {
        _store = store;
        _logger = logger;
    }

    public ValueTask CacheRequestAsync(OutputCacheContext context, CancellationToken cancellation)
    {
        var decision = _store.Resolve(context.HttpContext);
        context.HttpContext.Items[CacheContextKeys.Decision] = decision;

        if (!IsCacheableMethod(context.HttpContext.Request.Method) || !decision.Enabled)
        {
            context.EnableOutputCaching = false;
            context.AllowCacheLookup = false;
            context.AllowCacheStorage = false;
            return ValueTask.CompletedTask;
        }

        context.EnableOutputCaching = true;
        context.AllowCacheLookup = true;
        context.AllowCacheStorage = true;
        context.AllowLocking = true;
        context.ResponseExpirationTimeSpan = decision.Ttl;

        context.CacheVaryByRules.QueryKeys = CacheKeyBuilder.BuildQueryKeys(context.HttpContext, decision);

        return ValueTask.CompletedTask;
    }

    public ValueTask ServeFromCacheAsync(OutputCacheContext context, CancellationToken cancellation)
    {
        return ValueTask.CompletedTask;
    }

    public ValueTask ServeResponseAsync(OutputCacheContext context, CancellationToken cancellation)
    {
        var decision = GetDecision(context.HttpContext) ?? _store.Resolve(context.HttpContext);
        if (!IsCacheableMethod(context.HttpContext.Request.Method) || !decision.Enabled)
        {
            context.AllowCacheStorage = false;
            return ValueTask.CompletedTask;
        }

        var response = context.HttpContext.Response;
        if (response.StatusCode >= StatusCodes.Status400BadRequest)
        {
            context.AllowCacheStorage = false;
            return ValueTask.CompletedTask;
        }

        if (response.Headers.ContainsKey("Set-Cookie"))
        {
            context.AllowCacheStorage = false;
            return ValueTask.CompletedTask;
        }

        if (!decision.ForceCache && HasNoCacheHeader(response.Headers))
        {
            context.AllowCacheStorage = false;
            return ValueTask.CompletedTask;
        }

        context.ResponseExpirationTimeSpan = decision.Ttl;
        return ValueTask.CompletedTask;
    }

    private static bool IsCacheableMethod(string method)
    {
        return HttpMethods.IsGet(method) || HttpMethods.IsHead(method);
    }

    private static CacheDecision? GetDecision(HttpContext context)
    {
        if (context.Items.TryGetValue(CacheContextKeys.Decision, out var value) && value is CacheDecision decision)
        {
            return decision;
        }

        return null;
    }

    private static bool HasNoCacheHeader(IHeaderDictionary headers)
    {
        if (!headers.TryGetValue("Cache-Control", out var cacheControl))
        {
            return false;
        }

        var value = cacheControl.ToString();
        if (string.IsNullOrWhiteSpace(value))
        {
            return false;
        }

        return value.Contains("no-store", StringComparison.OrdinalIgnoreCase)
            || value.Contains("no-cache", StringComparison.OrdinalIgnoreCase)
            || value.Contains("private", StringComparison.OrdinalIgnoreCase);
    }
}

public static class CacheContextKeys
{
    public const string Decision = "cache.decision";
}
