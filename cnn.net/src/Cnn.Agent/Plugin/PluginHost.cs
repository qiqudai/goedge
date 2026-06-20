using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using Cnn.Agent.Config;
using Cnn.Agent.Logs;
using Microsoft.Extensions.Hosting;

namespace Cnn.Agent.Plugin;

public interface IPluginHost
{
    PluginDecision? Evaluate(HttpContext context);
    IReadOnlyCollection<PluginRuntimeState> GetStates();
}

public sealed record PluginRuntimeState(
    string Name,
    string Version,
    string ManifestPath,
    string AssemblyPath,
    PluginBreakerState BreakerState,
    string? LastError);

public sealed class PluginHost : BackgroundService, IPluginHost
{
    private const int DefaultMaxAssemblyBytes = 32 * 1024 * 1024;
    private static readonly StringComparison PathComparison = OperatingSystem.IsWindows()
        ? StringComparison.OrdinalIgnoreCase
        : StringComparison.Ordinal;

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly IConfiguration _configuration;
    private readonly AgentRuntimePaths _runtimePaths;
    private readonly ILogEventWriter _logWriter;
    private readonly ILogger<PluginHost> _logger;
    private readonly SemaphoreSlim _reloadLock = new(1, 1);
    private PluginRuntimeSnapshot _snapshot = PluginRuntimeSnapshot.Empty;

    public PluginHost(
        IConfiguration configuration,
        AgentRuntimePaths runtimePaths,
        ILogEventWriter logWriter,
        ILogger<PluginHost> logger)
    {
        _configuration = configuration;
        _runtimePaths = runtimePaths;
        _logWriter = logWriter;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await ReloadIfNeededAsync(force: false, stoppingToken);
            }
            catch (OperationCanceledException)
            {
                // ignore
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "plugin runtime reload loop failed");
            }

            var delaySec = Math.Max(1, ReadOptions().ScanIntervalSeconds);
            try
            {
                await Task.Delay(TimeSpan.FromSeconds(delaySec), stoppingToken);
            }
            catch (OperationCanceledException)
            {
                break;
            }
        }
    }

    public override async Task StopAsync(CancellationToken cancellationToken)
    {
        await base.StopAsync(cancellationToken);

        var previous = Volatile.Read(ref _snapshot);
        Volatile.Write(ref _snapshot, PluginRuntimeSnapshot.Empty);
        await DisposeEntriesAsync(previous.Entries);
    }

    public PluginDecision? Evaluate(HttpContext context)
    {
        var snapshot = Volatile.Read(ref _snapshot);
        if (!snapshot.Options.Enabled || snapshot.Entries.Count == 0)
        {
            return null;
        }

        var timeoutMs = Math.Max(1, snapshot.Options.EvalTimeoutMs);
        foreach (var entry in snapshot.Entries)
        {
            var now = DateTimeOffset.UtcNow;
            if (!entry.Breaker.TryEnter(now, out _))
            {
                continue;
            }

            using var cts = CancellationTokenSource.CreateLinkedTokenSource(context.RequestAborted);
            cts.CancelAfter(TimeSpan.FromMilliseconds(timeoutMs));

            try
            {
                var decision = entry.Plugin.EvaluateAsync(context, cts.Token).AsTask().ConfigureAwait(false).GetAwaiter().GetResult();
                entry.Breaker.RecordSuccess(DateTimeOffset.UtcNow);

                if (!decision.Handled)
                {
                    continue;
                }

                var statusCode = decision.Allowed
                    ? StatusCodes.Status200OK
                    : NormalizeBlockStatus(decision.StatusCode);
                var reason = string.IsNullOrWhiteSpace(decision.Reason)
                    ? $"plugin:{entry.Manifest.Name}"
                    : $"plugin:{entry.Manifest.Name}:{decision.Reason}";

                return new PluginDecision(true, decision.Allowed, statusCode, reason);
            }
            catch (OperationCanceledException) when (context.RequestAborted.IsCancellationRequested)
            {
                return null;
            }
            catch (OperationCanceledException)
            {
                entry.Breaker.RecordFailure(DateTimeOffset.UtcNow);
                entry.LastError = "timeout";
                WritePluginEvent("plugin_timeout", entry, "timeout");
                _logger.LogWarning("plugin evaluate timeout plugin={Plugin}", entry.Manifest.Name);
            }
            catch (Exception ex)
            {
                entry.Breaker.RecordFailure(DateTimeOffset.UtcNow);
                entry.LastError = ex.Message;
                WritePluginEvent("plugin_exception", entry, ex.Message);
                _logger.LogWarning(ex, "plugin evaluate failed plugin={Plugin}", entry.Manifest.Name);
            }
        }

        return null;
    }

    public IReadOnlyCollection<PluginRuntimeState> GetStates()
    {
        var snapshot = Volatile.Read(ref _snapshot);
        return snapshot.Entries
            .Select(entry => new PluginRuntimeState(
                Name: entry.Manifest.Name,
                Version: entry.Manifest.Version,
                ManifestPath: entry.ManifestPath,
                AssemblyPath: entry.AssemblyPath,
                BreakerState: entry.Breaker.GetState(DateTimeOffset.UtcNow),
                LastError: entry.LastError))
            .ToArray();
    }

    private async Task ReloadIfNeededAsync(bool force, CancellationToken cancellationToken)
    {
        var options = ReadOptions();
        if (!options.Enabled)
        {
            var current = Volatile.Read(ref _snapshot);
            if (!current.Options.Enabled && current.Entries.Count == 0)
            {
                return;
            }

            await SwapSnapshotAsync(new PluginRuntimeSnapshot(string.Empty, options, Array.Empty<PluginEntry>()), cancellationToken);
            _logger.LogInformation("plugin runtime disabled");
            return;
        }

        Directory.CreateDirectory(options.Directory);
        var manifests = DiscoverManifestFiles(options.Directory);
        var fingerprint = BuildFingerprint(options, manifests);
        var snapshot = Volatile.Read(ref _snapshot);

        if (!force && string.Equals(snapshot.Fingerprint, fingerprint, StringComparison.Ordinal))
        {
            return;
        }

        await _reloadLock.WaitAsync(cancellationToken);
        try
        {
            snapshot = Volatile.Read(ref _snapshot);
            if (!force && string.Equals(snapshot.Fingerprint, fingerprint, StringComparison.Ordinal))
            {
                return;
            }

            var loaded = await LoadEntriesAsync(manifests, options, cancellationToken);
            if (loaded.Count == 0 &&
                snapshot.Entries.Count > 0 &&
                HasEnabledManifest(manifests))
            {
                _logger.LogWarning(
                    "plugin runtime reload yielded zero entries while enabled manifests exist; keeping previous snapshot");
                return;
            }

            await SwapSnapshotAsync(new PluginRuntimeSnapshot(fingerprint, options, loaded), cancellationToken);

            _logger.LogInformation("plugin runtime reloaded entries={Count}", loaded.Count);
        }
        finally
        {
            _reloadLock.Release();
        }
    }

    private async Task SwapSnapshotAsync(PluginRuntimeSnapshot next, CancellationToken cancellationToken)
    {
        var previous = Volatile.Read(ref _snapshot);
        Volatile.Write(ref _snapshot, next);
        await DisposeEntriesAsync(previous.Entries);
        PersistState(next, cancellationToken);
    }

    private async Task<IReadOnlyList<PluginEntry>> LoadEntriesAsync(
        IReadOnlyList<string> manifests,
        PluginRuntimeOptions options,
        CancellationToken cancellationToken)
    {
        var entries = new List<PluginEntry>();
        foreach (var manifestPath in manifests)
        {
            cancellationToken.ThrowIfCancellationRequested();
            var loaded = await TryLoadEntryAsync(manifestPath, options, cancellationToken);
            if (loaded != null)
            {
                entries.Add(loaded);
            }
        }

        return entries;
    }

    private async Task<PluginEntry?> TryLoadEntryAsync(string manifestPath, PluginRuntimeOptions options, CancellationToken cancellationToken)
    {
        PluginLoadContext? loadContext = null;
        IRulePlugin? plugin = null;
        PluginManifest? manifest = null;
        string? assemblyPath = null;
        string? normalizedManifestPath = null;

        PluginEntry? Reject(string reason, string? detail = null)
        {
            _logger.LogWarning(
                "plugin rejected reason={Reason} name={Name} version={Version} path={Path} detail={Detail}",
                reason,
                manifest?.Name,
                manifest?.Version,
                manifestPath,
                detail);
            WritePluginLifecycleEvent(
                eventName: "plugin_rejected",
                name: manifest?.Name,
                version: manifest?.Version,
                reason: reason,
                message: detail,
                manifestPath: normalizedManifestPath ?? manifestPath,
                assemblyPath: assemblyPath);
            return null;
        }

        try
        {
            using (var stream = File.OpenRead(manifestPath))
            {
                manifest = await JsonSerializer.DeserializeAsync<PluginManifest>(stream, JsonOptions, cancellationToken);
            }

            if (manifest == null)
            {
                return Reject("manifest_parse_failed");
            }

            if (!manifest.Enable.GetValueOrDefault(true))
            {
                return null;
            }

            if (string.IsNullOrWhiteSpace(manifest.Name) ||
                string.IsNullOrWhiteSpace(manifest.Version) ||
                string.IsNullOrWhiteSpace(manifest.Sha256) ||
                string.IsNullOrWhiteSpace(manifest.EntryType))
            {
                return Reject("manifest_missing_required_fields");
            }

            if (options.AllowedPluginNames.Count > 0 &&
                !options.AllowedPluginNames.Contains(manifest.Name.Trim()))
            {
                return Reject("plugin_not_in_allowlist");
            }

            var pluginRoot = NormalizePath(options.Directory);
            normalizedManifestPath = NormalizePath(manifestPath);
            if (!IsPathUnderRoot(pluginRoot, normalizedManifestPath))
            {
                return Reject("manifest_outside_plugin_root");
            }

            var manifestDir = Path.GetDirectoryName(normalizedManifestPath);
            if (string.IsNullOrWhiteSpace(manifestDir))
            {
                return Reject("manifest_directory_invalid");
            }

            assemblyPath = ResolveAssemblyPath(normalizedManifestPath, manifest);
            if (string.IsNullOrWhiteSpace(assemblyPath) || !File.Exists(assemblyPath))
            {
                return Reject("assembly_not_found");
            }

            assemblyPath = NormalizePath(assemblyPath);
            if (!IsPathUnderRoot(pluginRoot, assemblyPath))
            {
                return Reject("assembly_outside_plugin_root");
            }

            if (options.RestrictAssemblyToPluginDirectory &&
                !IsPathUnderRoot(manifestDir, assemblyPath))
            {
                return Reject("assembly_outside_manifest_directory");
            }

            var info = new FileInfo(assemblyPath);
            if (!info.Exists || info.Length <= 0)
            {
                return Reject("assembly_invalid_size");
            }

            if (info.Length > options.MaxAssemblyBytes)
            {
                return Reject("assembly_too_large", $"{info.Length}>{options.MaxAssemblyBytes}");
            }

            var sha256 = ComputeSha256Hex(assemblyPath);
            if (!string.Equals(NormalizeHex(manifest.Sha256), sha256, StringComparison.OrdinalIgnoreCase))
            {
                return Reject("assembly_sha256_mismatch");
            }

            if (options.RequireSignature && !VerifySignature(manifest, assemblyPath, options))
            {
                return Reject("signature_verify_failed");
            }

            loadContext = new PluginLoadContext(assemblyPath);
            var assembly = loadContext.LoadFromAssemblyPath(assemblyPath);
            var pluginType = assembly.GetType(manifest.EntryType, throwOnError: false, ignoreCase: false);
            if (pluginType == null || !typeof(IRulePlugin).IsAssignableFrom(pluginType) || pluginType.IsAbstract)
            {
                loadContext.Unload();
                return Reject("plugin_type_invalid", manifest.EntryType);
            }

            if (Activator.CreateInstance(pluginType) is not IRulePlugin instance)
            {
                loadContext.Unload();
                return Reject("plugin_instance_create_failed", manifest.EntryType);
            }

            plugin = instance;

            using var initCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            initCts.CancelAfter(TimeSpan.FromSeconds(5));
            var settings = manifest.Settings ?? new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
            await plugin.InitializeAsync(settings, initCts.Token);

            var entry = new PluginEntry(
                manifest,
                manifestPath,
                assemblyPath,
                loadContext,
                plugin,
                new PluginCircuitBreaker(options.Breaker));

            WritePluginLifecycleEvent(
                eventName: "plugin_loaded",
                name: manifest.Name,
                version: manifest.Version,
                reason: "loaded",
                message: null,
                manifestPath: normalizedManifestPath,
                assemblyPath: assemblyPath);

            return entry;
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "plugin load failed path={Path}", manifestPath);
            WritePluginLifecycleEvent(
                eventName: "plugin_rejected",
                name: manifest?.Name,
                version: manifest?.Version,
                reason: "load_exception",
                message: ex.Message,
                manifestPath: normalizedManifestPath ?? manifestPath,
                assemblyPath: assemblyPath);

            if (plugin != null)
            {
                try
                {
                    await plugin.DisposeAsync();
                }
                catch
                {
                    // ignore
                }
            }

            if (loadContext != null)
            {
                try
                {
                    loadContext.Unload();
                }
                catch
                {
                    // ignore
                }
            }

            return null;
        }
    }

    private async Task DisposeEntriesAsync(IReadOnlyList<PluginEntry> entries)
    {
        foreach (var entry in entries)
        {
            try
            {
                await entry.DisposeAsync();
            }
            catch
            {
                // ignore
            }
        }
    }

    private PluginRuntimeOptions ReadOptions()
    {
        var section = _configuration.GetSection("Plugins");
        var breakerSection = section.GetSection("breaker");

        var options = new PluginRuntimeOptions
        {
            Enabled = ReadBool(section, false, "enabled", "Enabled"),
            Directory = ResolveDirectory(ReadString(section, "plugins", "directory", "Directory")),
            RequireSignature = ReadBool(section, true, "require_signature", "RequireSignature"),
            MaxAssemblyBytes = ReadInt(section, DefaultMaxAssemblyBytes, "max_assembly_bytes", "MaxAssemblyBytes"),
            RestrictAssemblyToPluginDirectory = ReadBool(
                section,
                true,
                "restrict_assembly_to_plugin_directory",
                "RestrictAssemblyToPluginDirectory"),
            EvalTimeoutMs = ReadInt(section, 5, "eval_timeout_ms", "EvalTimeoutMs"),
            ScanIntervalSeconds = ReadInt(section, 3, "scan_interval_seconds", "ScanIntervalSeconds"),
            SignaturePublicKeyPath = ResolveOptionalPath(ReadString(section, null, "signature_public_key_path", "SignaturePublicKeyPath")),
            SignatureHmacKey = ReadString(section, null, "signature_hmac_key", "SignatureHmacKey"),
            AllowedPluginNames = ReadStringSet(section, "allowed_plugin_names", "AllowedPluginNames", "allow_names", "AllowNames"),
            Breaker = new PluginBreakerOptions
            {
                FailThreshold = ReadInt(breakerSection, 20, "fail_threshold", "FailThreshold"),
                WindowSeconds = ReadInt(breakerSection, 60, "window_seconds", "WindowSeconds"),
                OpenSeconds = ReadInt(breakerSection, 120, "open_seconds", "OpenSeconds")
            }
        };

        if (options.EvalTimeoutMs <= 0)
        {
            options.EvalTimeoutMs = 5;
        }

        if (options.MaxAssemblyBytes <= 0)
        {
            options.MaxAssemblyBytes = DefaultMaxAssemblyBytes;
        }

        return options;
    }

    private string ResolveDirectory(string? raw)
    {
        var path = raw?.Trim() ?? "plugins";
        if (path.Length == 0)
        {
            path = "plugins";
        }

        if (Path.IsPathRooted(path))
        {
            return path;
        }

        return Path.Combine(_runtimePaths.RuntimeRoot, path);
    }

    private string? ResolveOptionalPath(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        if (Path.IsPathRooted(raw))
        {
            return raw;
        }

        return Path.Combine(_runtimePaths.RuntimeRoot, raw);
    }

    private static IReadOnlyList<string> DiscoverManifestFiles(string directory)
    {
        var set = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var path in Directory.EnumerateFiles(directory, "manifest.json", SearchOption.AllDirectories))
        {
            set.Add(path);
        }

        foreach (var path in Directory.EnumerateFiles(directory, "*.manifest.json", SearchOption.AllDirectories))
        {
            set.Add(path);
        }

        return set.OrderBy(static s => s, StringComparer.Ordinal).ToArray();
    }

    private static bool HasEnabledManifest(IReadOnlyList<string> manifests)
    {
        foreach (var path in manifests)
        {
            try
            {
                using var stream = File.OpenRead(path);
                var manifest = JsonSerializer.Deserialize<PluginManifest>(stream, JsonOptions);
                if (manifest == null)
                {
                    continue;
                }

                if (manifest.Enable.GetValueOrDefault(true))
                {
                    return true;
                }
            }
            catch
            {
                // treat parse/read failures as potentially enabled and keep old snapshot
                return true;
            }
        }

        return false;
    }

    private static string BuildFingerprint(PluginRuntimeOptions options, IReadOnlyList<string> manifests)
    {
        var sb = new StringBuilder(256);
        sb.Append(options.Enabled).Append('|')
            .Append(options.Directory).Append('|')
            .Append(options.RequireSignature).Append('|')
            .Append(options.MaxAssemblyBytes).Append('|')
            .Append(options.RestrictAssemblyToPluginDirectory).Append('|')
            .Append(options.EvalTimeoutMs).Append('|')
            .Append(options.ScanIntervalSeconds).Append('|')
            .Append(options.SignaturePublicKeyPath).Append('|')
            .Append(ComputeTextFingerprint(options.SignatureHmacKey)).Append('|')
            .Append(options.Breaker.FailThreshold).Append('|')
            .Append(options.Breaker.WindowSeconds).Append('|')
            .Append(options.Breaker.OpenSeconds).Append('|')
            .Append(manifests.Count);

        foreach (var allowed in options.AllowedPluginNames.OrderBy(static v => v, StringComparer.OrdinalIgnoreCase))
        {
            sb.Append("|allow:").Append(allowed);
        }

        if (!string.IsNullOrWhiteSpace(options.SignaturePublicKeyPath))
        {
            sb.Append("|public-key:");
            AppendFileFingerprint(sb, options.SignaturePublicKeyPath);
        }

        foreach (var path in manifests)
        {
            sb.Append('|')
                .Append(path);
            AppendFileFingerprint(sb, path);

            var assemblyPath = TryResolveAssemblyPathFromManifest(path);
            if (!string.IsNullOrWhiteSpace(assemblyPath))
            {
                sb.Append('|')
                    .Append("asm:")
                    .Append(assemblyPath);
                AppendFileFingerprint(sb, assemblyPath);
            }
        }

        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes(sb.ToString()));
        return Convert.ToHexString(bytes);
    }

    private static string ComputeTextFingerprint(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return string.Empty;
        }

        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes(value));
        return Convert.ToHexString(bytes);
    }

    private static void AppendFileFingerprint(StringBuilder sb, string path)
    {
        try
        {
            var info = new FileInfo(path);
            if (!info.Exists)
            {
                sb.Append(":missing");
                return;
            }

            sb.Append(':')
                .Append(info.Length)
                .Append(':')
                .Append(info.LastWriteTimeUtc.Ticks)
                .Append(':')
                .Append(ComputeSha256Hex(path));
        }
        catch
        {
            sb.Append(":error");
        }
    }

    private static string? TryResolveAssemblyPathFromManifest(string manifestPath)
    {
        try
        {
            using var stream = File.OpenRead(manifestPath);
            var manifest = JsonSerializer.Deserialize<PluginManifest>(stream, JsonOptions);
            if (manifest == null)
            {
                return null;
            }

            return ResolveAssemblyPath(manifestPath, manifest);
        }
        catch
        {
            return null;
        }
    }

    private static string? ResolveAssemblyPath(string manifestPath, PluginManifest manifest)
    {
        var manifestDir = Path.GetDirectoryName(manifestPath);
        if (string.IsNullOrWhiteSpace(manifestDir))
        {
            return null;
        }

        if (!string.IsNullOrWhiteSpace(manifest.EntryAssembly))
        {
            var assemblyPath = manifest.EntryAssembly!.Trim();
            if (Path.IsPathRooted(assemblyPath))
            {
                return assemblyPath;
            }

            return Path.Combine(manifestDir, assemblyPath);
        }

        var dlls = Directory.EnumerateFiles(manifestDir, "*.dll", SearchOption.TopDirectoryOnly)
            .OrderBy(static p => p, StringComparer.OrdinalIgnoreCase)
            .ToArray();
        if (dlls.Length == 0)
        {
            return null;
        }

        return dlls[0];
    }

    private bool VerifySignature(PluginManifest manifest, string assemblyPath, PluginRuntimeOptions options)
    {
        var signature = manifest.Signature?.Trim();
        if (string.IsNullOrWhiteSpace(signature))
        {
            return false;
        }

        var assemblyBytes = File.ReadAllBytes(assemblyPath);

        var publicKeyPath = options.SignaturePublicKeyPath?.Trim();
        if (!string.IsNullOrWhiteSpace(publicKeyPath) && File.Exists(publicKeyPath))
        {
            var signatureBytes = TryDecodeSignature(signature);
            if (signatureBytes == null || signatureBytes.Length == 0)
            {
                return false;
            }

            var pem = File.ReadAllText(publicKeyPath);
            using var rsa = RSA.Create();
            rsa.ImportFromPem(pem);
            return rsa.VerifyData(assemblyBytes, signatureBytes, HashAlgorithmName.SHA256, RSASignaturePadding.Pkcs1);
        }

        var hmacKey = options.SignatureHmacKey?.Trim();
        if (!string.IsNullOrWhiteSpace(hmacKey))
        {
            using var hmac = new HMACSHA256(Encoding.UTF8.GetBytes(hmacKey));
            var mac = hmac.ComputeHash(assemblyBytes);
            return string.Equals(Convert.ToHexString(mac), NormalizeHex(signature), StringComparison.OrdinalIgnoreCase);
        }

        return false;
    }

    private static byte[]? TryDecodeSignature(string text)
    {
        try
        {
            return Convert.FromBase64String(text);
        }
        catch
        {
            // ignore
        }

        try
        {
            return Convert.FromHexString(NormalizeHex(text));
        }
        catch
        {
            return null;
        }
    }

    private static string ComputeSha256Hex(string filePath)
    {
        using var stream = File.OpenRead(filePath);
        var hash = SHA256.HashData(stream);
        return Convert.ToHexString(hash);
    }

    private static string NormalizeHex(string value)
    {
        return value.Trim().Replace("-", string.Empty, StringComparison.Ordinal).ToUpperInvariant();
    }

    private static string NormalizePath(string path)
    {
        var full = Path.GetFullPath(path);
        var trimmed = full.TrimEnd(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar);
        if (trimmed.Length > 0)
        {
            return trimmed;
        }

        var root = Path.GetPathRoot(full);
        return string.IsNullOrWhiteSpace(root) ? full : root;
    }

    private static bool IsPathUnderRoot(string rootPath, string targetPath)
    {
        var root = NormalizePath(rootPath);
        var target = NormalizePath(targetPath);

        if (string.Equals(root, target, PathComparison))
        {
            return true;
        }

        return target.StartsWith(root + Path.DirectorySeparatorChar, PathComparison) ||
               target.StartsWith(root + Path.AltDirectorySeparatorChar, PathComparison);
    }

    private static int NormalizeBlockStatus(int code)
    {
        return code >= 400 ? code : StatusCodes.Status403Forbidden;
    }

    private static string? ReadString(IConfiguration section, string? fallback, params string[] keys)
    {
        foreach (var key in keys)
        {
            var value = section[key];
            if (!string.IsNullOrWhiteSpace(value))
            {
                return value.Trim();
            }
        }

        return fallback;
    }

    private static int ReadInt(IConfiguration section, int fallback, params string[] keys)
    {
        foreach (var key in keys)
        {
            var raw = section[key];
            if (int.TryParse(raw, out var value))
            {
                return value;
            }
        }

        return fallback;
    }

    private static bool ReadBool(IConfiguration section, bool fallback, params string[] keys)
    {
        foreach (var key in keys)
        {
            var raw = section[key];
            if (bool.TryParse(raw, out var value))
            {
                return value;
            }
        }

        return fallback;
    }

    private static HashSet<string> ReadStringSet(IConfiguration section, params string[] keys)
    {
        var set = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var key in keys)
        {
            var raw = section[key];
            if (!string.IsNullOrWhiteSpace(raw))
            {
                foreach (var token in raw.Split([',', ';', '|'], StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries))
                {
                    if (!string.IsNullOrWhiteSpace(token))
                    {
                        set.Add(token.Trim());
                    }
                }
            }

            foreach (var item in section.GetSection(key).GetChildren())
            {
                var value = item.Value;
                if (!string.IsNullOrWhiteSpace(value))
                {
                    set.Add(value.Trim());
                }
            }
        }

        return set;
    }

    private void PersistState(PluginRuntimeSnapshot snapshot, CancellationToken cancellationToken)
    {
        try
        {
            cancellationToken.ThrowIfCancellationRequested();

            var payload = new
            {
                updated_at = DateTimeOffset.UtcNow,
                enabled = snapshot.Options.Enabled,
                policy = new
                {
                    require_signature = snapshot.Options.RequireSignature,
                    max_assembly_bytes = snapshot.Options.MaxAssemblyBytes,
                    restrict_to_plugin_directory = snapshot.Options.RestrictAssemblyToPluginDirectory,
                    allowed_plugins = snapshot.Options.AllowedPluginNames.OrderBy(static v => v, StringComparer.OrdinalIgnoreCase).ToArray()
                },
                plugin_count = snapshot.Entries.Count,
                plugins = snapshot.Entries.Select(entry => new
                {
                    name = entry.Manifest.Name,
                    version = entry.Manifest.Version,
                    manifest = entry.ManifestPath,
                    assembly = entry.AssemblyPath,
                    breaker = entry.Breaker.GetState(DateTimeOffset.UtcNow).ToString(),
                    last_error = entry.LastError
                }).ToArray()
            };

            Directory.CreateDirectory(_runtimePaths.ConfDir);
            var tempPath = _runtimePaths.PluginStatePath + ".tmp";
            File.WriteAllText(tempPath, JsonSerializer.Serialize(payload, new JsonSerializerOptions(JsonSerializerDefaults.Web)
            {
                WriteIndented = true
            }));
            File.Move(tempPath, _runtimePaths.PluginStatePath, true);
        }
        catch
        {
            // ignore persist failures
        }
    }

    private void WritePluginEvent(string eventName, PluginEntry entry, string? message)
    {
        WritePluginLifecycleEvent(
            eventName,
            entry.Manifest.Name,
            entry.Manifest.Version,
            reason: eventName,
            message: message,
            manifestPath: entry.ManifestPath,
            assemblyPath: entry.AssemblyPath);
    }

    private void WritePluginLifecycleEvent(
        string eventName,
        string? name,
        string? version,
        string? reason,
        string? message,
        string? manifestPath,
        string? assemblyPath)
    {
        _ = _logWriter.TryWrite(new LogEvent(
            DateTimeOffset.UtcNow,
            LogChannels.System,
            "warning",
            eventName,
            Guid.NewGuid().ToString("N"),
            new Dictionary<string, object?>
            {
                ["plugin"] = name,
                ["version"] = version,
                ["reason"] = reason,
                ["message"] = message,
                ["manifest"] = manifestPath,
                ["assembly"] = assemblyPath
            }));
    }

    private sealed class PluginEntry(
        PluginManifest manifest,
        string manifestPath,
        string assemblyPath,
        PluginLoadContext loadContext,
        IRulePlugin plugin,
        PluginCircuitBreaker breaker)
    {
        public PluginManifest Manifest { get; } = manifest;
        public string ManifestPath { get; } = manifestPath;
        public string AssemblyPath { get; } = assemblyPath;
        public PluginLoadContext LoadContext { get; } = loadContext;
        public IRulePlugin Plugin { get; } = plugin;
        public PluginCircuitBreaker Breaker { get; } = breaker;
        public string? LastError { get; set; }

        public async ValueTask DisposeAsync()
        {
            try
            {
                await Plugin.DisposeAsync();
            }
            catch
            {
                // ignore plugin dispose errors
            }

            try
            {
                LoadContext.Unload();
            }
            catch
            {
                // ignore unload errors
            }
        }
    }

    private sealed class PluginRuntimeSnapshot(string fingerprint, PluginRuntimeOptions options, IReadOnlyList<PluginEntry> entries)
    {
        public static PluginRuntimeSnapshot Empty { get; } = new(
            string.Empty,
            new PluginRuntimeOptions { Enabled = false },
            Array.Empty<PluginEntry>());

        public string Fingerprint { get; } = fingerprint;
        public PluginRuntimeOptions Options { get; } = options;
        public IReadOnlyList<PluginEntry> Entries { get; } = entries;
    }
}
