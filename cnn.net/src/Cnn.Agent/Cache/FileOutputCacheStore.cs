using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.OutputCaching;
using Microsoft.Extensions.Options;

namespace Cnn.Agent.Cache;

public sealed class FileOutputCacheStore : IOutputCacheStore
{
    private readonly IHttpContextAccessor _httpContextAccessor;
    private readonly CacheRuntimeStore _store;
    private readonly CacheOptions _options;

    public FileOutputCacheStore(IOptions<CacheOptions> options, IHttpContextAccessor httpContextAccessor, CacheRuntimeStore store)
    {
        _options = options.Value;
        _httpContextAccessor = httpContextAccessor;
        _store = store;
    }

    public ValueTask<byte[]?> GetAsync(string key, CancellationToken cancellationToken)
    {
        var (path, metaPath) = GetPathsFromContext();
        if (path == null || metaPath == null)
        {
            return ValueTask.FromResult<byte[]?>(null);
        }

        if (!File.Exists(path) || !File.Exists(metaPath))
        {
            return ValueTask.FromResult<byte[]?>(null);
        }

        var expireAt = ReadExpireAt(metaPath);
        if (expireAt.HasValue && expireAt.Value <= DateTimeOffset.UtcNow)
        {
            TryDelete(path, metaPath);
            return ValueTask.FromResult<byte[]?>(null);
        }

        return ValueTask.FromResult<byte[]?>(File.ReadAllBytes(path));
    }

    public ValueTask SetAsync(string key, byte[] value, string[]? tags, TimeSpan validFor, CancellationToken cancellationToken)
    {
        var (path, metaPath) = GetPathsFromContext();
        if (path == null || metaPath == null)
        {
            return ValueTask.CompletedTask;
        }

        var dir = Path.GetDirectoryName(path);
        if (!string.IsNullOrWhiteSpace(dir))
        {
            Directory.CreateDirectory(dir);
        }

        File.WriteAllBytes(path, value);

        var expireAt = DateTimeOffset.UtcNow.Add(validFor <= TimeSpan.Zero ? TimeSpan.FromSeconds(1) : validFor);
        File.WriteAllText(metaPath, expireAt.ToUnixTimeSeconds().ToString());

        return ValueTask.CompletedTask;
    }

    public ValueTask EvictByTagAsync(string tag, CancellationToken cancellationToken)
    {
        return ValueTask.CompletedTask;
    }

    private (string? DataPath, string? MetaPath) GetPathsFromContext()
    {
        var context = _httpContextAccessor.HttpContext;
        if (context == null)
        {
            return (null, null);
        }

        var decision = GetDecision(context) ?? _store.Resolve(context);
        context.Items[CacheContextKeys.Decision] = decision;

        var key = CacheKeyBuilder.BuildRelativeKey(context, decision);
        if (string.IsNullOrWhiteSpace(key))
        {
            return (null, null);
        }

        var dataPath = Path.Combine(_options.Root, key.Replace('/', Path.DirectorySeparatorChar));
        var metaPath = dataPath + ".meta";
        return (dataPath, metaPath);
    }

    private static CacheDecision? GetDecision(HttpContext context)
    {
        if (context.Items.TryGetValue(CacheContextKeys.Decision, out var value) && value is CacheDecision decision)
        {
            return decision;
        }

        return null;
    }

    private static DateTimeOffset? ReadExpireAt(string metaPath)
    {
        try
        {
            var text = File.ReadAllText(metaPath).Trim();
            if (long.TryParse(text, out var seconds))
            {
                return DateTimeOffset.FromUnixTimeSeconds(seconds);
            }
        }
        catch
        {
            return null;
        }

        return null;
    }

    private static void TryDelete(string dataPath, string metaPath)
    {
        try
        {
            File.Delete(dataPath);
        }
        catch
        {
            // ignore
        }

        try
        {
            File.Delete(metaPath);
        }
        catch
        {
            // ignore
        }
    }
}
