using System.Security.Cryptography;
using System.Text;
using Cnn.Agent.Logs;
using Cnn.Common.Contracts.Agent;
using Microsoft.Extensions.Options;

namespace Cnn.Agent.Stream;

public interface IStreamRuntime
{
    StreamApplyResult Apply(EdgeConfigDto config);
    IReadOnlyCollection<StreamListenerState> GetStates();
    StreamRuntimeReport GetReport();
}

public sealed class StreamRuntime : IStreamRuntime
{
    private readonly StreamRouteCompiler _compiler;
    private readonly ILogEventWriter _logWriter;
    private readonly KernelNatRuntime _kernelNatRuntime;
    private readonly IOptionsMonitor<StreamRuntimeOptions> _optionsMonitor;
    private readonly ILoggerFactory _loggerFactory;
    private readonly ILogger<StreamRuntime> _logger;
    private readonly SemaphoreSlim _applyLock = new(1, 1);
    private readonly object _stateLock = new();
    private readonly Dictionary<string, ListenerEntry> _listeners = new(StringComparer.OrdinalIgnoreCase);
    private string _lastPlanHash = string.Empty;
    private bool _natActive;
    private string _activeMode = "userspace";
    private string? _lastError;
    private long _lastConfigVersion;
    private int _lastReceived;
    private int _lastPlanned;
    private int _lastApplied;
    private int _lastSkipped;
    private IReadOnlyList<string> _lastSkipReasons = Array.Empty<string>();

    public StreamRuntime(
        StreamRouteCompiler compiler,
        ILogEventWriter logWriter,
        KernelNatRuntime kernelNatRuntime,
        IOptionsMonitor<StreamRuntimeOptions> optionsMonitor,
        ILoggerFactory loggerFactory,
        ILogger<StreamRuntime> logger)
    {
        _compiler = compiler;
        _logWriter = logWriter;
        _kernelNatRuntime = kernelNatRuntime;
        _optionsMonitor = optionsMonitor;
        _loggerFactory = loggerFactory;
        _logger = logger;
    }

    public StreamApplyResult Apply(EdgeConfigDto config)
    {
        if (config == null)
        {
            var errors = new[] { "stream config is null" };
            UpdateLastApplySummary(
                configVersion: 0,
                received: 0,
                planned: 0,
                applied: 0,
                skipped: 0,
                skipReasons: errors);
            return new StreamApplyResult(false, 0, 0, 0, errors);
        }

        _applyLock.Wait();
        try
        {
            var options = _optionsMonitor.CurrentValue ?? new StreamRuntimeOptions();
            var mode = NormalizeMode(options.Mode);
            var (plans, compileErrors) = _compiler.Compile(config);
            var received = config.Streams?.Count ?? 0;
            var planned = plans.Count;
            var skipped = Math.Max(0, received - plans.Select(static p => p.StreamId).Distinct().Count());
            var planHash = $"{mode}:{BuildPlanHash(plans)}";
            if (string.Equals(planHash, _lastPlanHash, StringComparison.Ordinal))
            {
                var skipReasons = BuildSkipReasons(compileErrors, "plan_unchanged");
                UpdateLastApplySummary(
                    configVersion: config.Version,
                    received: received,
                    planned: planned,
                    applied: _lastApplied,
                    skipped: Math.Max(skipped, received),
                    skipReasons: skipReasons);
                return new StreamApplyResult(
                    compileErrors.Count == 0,
                    Started: 0,
                    Stopped: 0,
                    Restarted: 0,
                    Errors: compileErrors,
                    Received: received,
                    Planned: planned,
                    Applied: _lastApplied,
                    Skipped: Math.Max(skipped, received),
                    SkipReasons: skipReasons);
            }

            if (string.Equals(mode, "nat", StringComparison.Ordinal))
            {
                var natResult = _kernelNatRuntime.Apply(plans, options);
                if (natResult.Success)
                {
                    var stopped = StopAllListeners();
                    _natActive = true;
                    _activeMode = "nat";
                    _lastError = compileErrors.Count > 0 ? compileErrors[0] : null;
                    _lastPlanHash = planHash;
                    var errors = new List<string>(compileErrors);
                    var skipReasons = BuildSkipReasons(compileErrors);
                    UpdateLastApplySummary(
                        configVersion: config.Version,
                        received: received,
                        planned: planned,
                        applied: natResult.RuleCount,
                        skipped: skipped,
                        skipReasons: skipReasons);
                    _logger.LogInformation(
                        "stream nat apply finished version={Version} received={Received} planned={Planned} rules={RuleCount} skipped={Skipped} stopped={Stopped} errors={Errors}",
                        config.Version,
                        received,
                        planned,
                        natResult.RuleCount,
                        skipped,
                        stopped,
                        errors.Count);

                    return new StreamApplyResult(
                        errors.Count == 0,
                        Started: 0,
                        Stopped: stopped,
                        Restarted: 0,
                        Errors: errors,
                        Received: received,
                        Planned: planned,
                        Applied: natResult.RuleCount,
                        Skipped: skipped,
                        SkipReasons: skipReasons);
                }

                var fallbackEnabled = options.FallbackToUserspaceOnNatFailure;
                _logger.LogWarning(
                    "stream nat apply failed version={Version} fallback={Fallback} errors={Errors}",
                    config.Version,
                    fallbackEnabled,
                    string.Join("; ", natResult.Errors.Take(3)));

                if (!fallbackEnabled)
                {
                    var natErrors = new List<string>(compileErrors);
                    natErrors.AddRange(natResult.Errors);
                    _lastError = natErrors.Count > 0 ? natErrors[0] : null;
                    var skipReasons = BuildSkipReasons(natErrors);
                    UpdateLastApplySummary(
                        configVersion: config.Version,
                        received: received,
                        planned: planned,
                        applied: 0,
                        skipped: Math.Max(skipped, received),
                        skipReasons: skipReasons);
                    return new StreamApplyResult(
                        Success: false,
                        Started: 0,
                        Stopped: 0,
                        Restarted: 0,
                        Errors: natErrors,
                        Received: received,
                        Planned: planned,
                        Applied: 0,
                        Skipped: Math.Max(skipped, received),
                        SkipReasons: skipReasons);
                }
            }

            EnsureNatCleared(options);
            var applyResult = ApplyUserspace(config.Version, received, skipped, plans, compileErrors);
            _activeMode = "userspace";
            _lastError = applyResult.Errors.Count > 0 ? applyResult.Errors[0] : null;
            _lastPlanHash = planHash;
            return applyResult;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "stream apply failed version={Version}", config.Version);
            _lastError = ex.Message;
            var errors = new[] { ex.Message };
            UpdateLastApplySummary(
                configVersion: config.Version,
                received: config.Streams?.Count ?? 0,
                planned: 0,
                applied: 0,
                skipped: config.Streams?.Count ?? 0,
                skipReasons: errors);
            return new StreamApplyResult(false, 0, 0, 0, errors);
        }
        finally
        {
            _applyLock.Release();
        }
    }

    public IReadOnlyCollection<StreamListenerState> GetStates()
    {
        if (_natActive)
        {
            return _kernelNatRuntime.GetStates();
        }

        List<StreamListenerState> states;
        lock (_stateLock)
        {
            states = _listeners.Values.Select(v => v.Listener.GetState()).ToList();
        }

        return states;
    }

    public StreamRuntimeReport GetReport()
    {
        var options = _optionsMonitor.CurrentValue ?? new StreamRuntimeOptions();
        var configured = NormalizeMode(options.Mode);
        return new StreamRuntimeReport(
            ConfiguredMode: configured,
            ActiveMode: _activeMode,
            NatActive: _natActive,
            LastError: _lastError,
            LastConfigVersion: _lastConfigVersion,
            LastReceived: _lastReceived,
            LastPlanned: _lastPlanned,
            LastApplied: _lastApplied,
            LastSkipped: _lastSkipped,
            LastSkipReasons: _lastSkipReasons,
            States: GetStates());
    }

    private StreamApplyResult ApplyUserspace(
        long version,
        int received,
        int skipped,
        IReadOnlyList<StreamListenerPlan> plans,
        IReadOnlyList<string> compileErrors)
    {
        var errors = new List<string>(compileErrors);

        var planByKey = plans.ToDictionary(p => p.Key, p => p, StringComparer.OrdinalIgnoreCase);

        var removed = new List<ListenerEntry>();
        lock (_stateLock)
        {
            var keys = _listeners.Keys.ToArray();
            foreach (var key in keys)
            {
                if (planByKey.ContainsKey(key))
                {
                    continue;
                }

                removed.Add(_listeners[key]);
                _listeners.Remove(key);
            }
        }

        var stopped = 0;
        foreach (var entry in removed)
        {
            StopListener(entry.Listener);
            stopped++;
        }

        var started = 0;
        var restarted = 0;
        foreach (var plan in plans)
        {
            var signature = StreamListener.BuildSignature(plan);
            ListenerEntry? existing = null;
            lock (_stateLock)
            {
                _listeners.TryGetValue(plan.Key, out existing);
            }

            if (existing != null && string.Equals(existing.Signature, signature, StringComparison.Ordinal))
            {
                continue;
            }

            if (existing != null)
            {
                lock (_stateLock)
                {
                    _listeners.Remove(plan.Key);
                }

                StopListener(existing.Listener);
            }

            var listener = new StreamListener(plan, _logWriter, _loggerFactory.CreateLogger<StreamListener>());
            var startedOk = StartListener(listener);
            if (!startedOk)
            {
                errors.Add($"stream listener start failed: {plan.Key}");
                continue;
            }

            lock (_stateLock)
            {
                _listeners[plan.Key] = new ListenerEntry(signature, listener);
            }

            if (existing == null)
            {
                started++;
            }
            else
            {
                restarted++;
            }
        }

        var success = errors.Count == 0;
        var applied = GetCurrentListenerCount();
        var skipReasons = BuildSkipReasons(compileErrors);
        UpdateLastApplySummary(
            configVersion: version,
            received: received,
            planned: plans.Count,
            applied: applied,
            skipped: skipped,
            skipReasons: skipReasons);
        _logger.LogInformation(
            "stream apply finished version={Version} received={Received} planned={Planned} applied={Applied} skipped={Skipped} started={Started} stopped={Stopped} restarted={Restarted} errors={Errors}",
            version,
            received,
            plans.Count,
            applied,
            skipped,
            started,
            stopped,
            restarted,
            errors.Count);

        _ = _logWriter.TryWrite(new LogEvent(
            DateTimeOffset.UtcNow,
            LogChannels.System,
            success ? "information" : "warning",
            "stream_apply",
            Guid.NewGuid().ToString("N"),
            new Dictionary<string, object?>
            {
                ["config_version"] = version,
                ["received"] = received,
                ["planned"] = plans.Count,
                ["applied"] = applied,
                ["skipped"] = skipped,
                ["started"] = started,
                ["stopped"] = stopped,
                ["restarted"] = restarted,
                ["errors"] = errors,
                ["skip_reasons"] = skipReasons
            }));

        return new StreamApplyResult(
            Success: success,
            Started: started,
            Stopped: stopped,
            Restarted: restarted,
            Errors: errors,
            Received: received,
            Planned: plans.Count,
            Applied: applied,
            Skipped: skipped,
            SkipReasons: skipReasons);
    }

    private int StopAllListeners()
    {
        List<ListenerEntry> all;
        lock (_stateLock)
        {
            all = _listeners.Values.ToList();
            _listeners.Clear();
        }

        foreach (var entry in all)
        {
            StopListener(entry.Listener);
        }

        return all.Count;
    }

    private void EnsureNatCleared(StreamRuntimeOptions options)
    {
        if (!_natActive)
        {
            return;
        }

        if (!_kernelNatRuntime.Clear(options, out var clearError))
        {
            _logger.LogWarning("stream nat clear failed: {Error}", clearError ?? "unknown");
        }

        _natActive = false;
    }

    private static string NormalizeMode(string? raw)
    {
        return raw?.Trim().ToLowerInvariant() switch
        {
            "nat" => "nat",
            _ => "userspace"
        };
    }

    private static bool StartListener(StreamListener listener)
    {
        try
        {
            return listener.StartAsync(CancellationToken.None).ConfigureAwait(false).GetAwaiter().GetResult();
        }
        catch
        {
            return false;
        }
    }

    private static void StopListener(StreamListener listener)
    {
        try
        {
            listener.StopAsync(CancellationToken.None).ConfigureAwait(false).GetAwaiter().GetResult();
        }
        catch
        {
            // ignore stop errors
        }
    }

    private static string BuildPlanHash(IReadOnlyList<StreamListenerPlan> plans)
    {
        if (plans.Count == 0)
        {
            return "empty";
        }

        var signatures = plans
            .Select(StreamListener.BuildSignature)
            .OrderBy(static s => s, StringComparer.Ordinal)
            .ToArray();

        var joined = string.Join('\n', signatures);
        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes(joined));
        return Convert.ToHexString(bytes);
    }

    private int GetCurrentListenerCount()
    {
        lock (_stateLock)
        {
            return _listeners.Count;
        }
    }

    private void UpdateLastApplySummary(
        long configVersion,
        int received,
        int planned,
        int applied,
        int skipped,
        IReadOnlyList<string> skipReasons)
    {
        _lastConfigVersion = configVersion;
        _lastReceived = Math.Max(0, received);
        _lastPlanned = Math.Max(0, planned);
        _lastApplied = Math.Max(0, applied);
        _lastSkipped = Math.Max(0, skipped);
        _lastSkipReasons = skipReasons.Count == 0 ? Array.Empty<string>() : skipReasons.ToArray();
    }

    private static IReadOnlyList<string> BuildSkipReasons(IReadOnlyList<string> compileErrors, params string[] extraReasons)
    {
        var reasons = new List<string>();
        reasons.AddRange(extraReasons.Where(static r => !string.IsNullOrWhiteSpace(r)).Select(static r => r.Trim()));
        if (compileErrors.Count > 0)
        {
            reasons.Add("compile_errors");
        }

        if (reasons.Count == 0)
        {
            return Array.Empty<string>();
        }

        return reasons
            .Distinct(StringComparer.OrdinalIgnoreCase)
            .ToArray();
    }

    private sealed record ListenerEntry(string Signature, StreamListener Listener);
}
