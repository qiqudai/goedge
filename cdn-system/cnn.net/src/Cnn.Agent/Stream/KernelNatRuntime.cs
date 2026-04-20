using System.Diagnostics;
using System.Net;

namespace Cnn.Agent.Stream;

public sealed class KernelNatRuntime
{
    private const string NatTable = "nat";
    private const string NatChain = "CNN_STREAM_DNAT";
    private readonly ILogger<KernelNatRuntime> _logger;
    private readonly object _stateLock = new();
    private IReadOnlyCollection<StreamListenerState> _states = Array.Empty<StreamListenerState>();

    public KernelNatRuntime(ILogger<KernelNatRuntime> logger)
    {
        _logger = logger;
    }

    public NatApplyResult Apply(IReadOnlyList<StreamListenerPlan> plans, StreamRuntimeOptions options)
    {
        if (!OperatingSystem.IsLinux())
        {
            return NatApplyResult.Fail("nat mode is only supported on linux");
        }

        var errors = new List<string>();
        var rules = BuildRules(plans, errors);
        if (errors.Count > 0)
        {
            return NatApplyResult.Fail(errors);
        }

        var binary = string.IsNullOrWhiteSpace(options.IptablesBinary) ? "iptables" : options.IptablesBinary.Trim();
        var timeoutMs = options.CommandTimeoutMs > 0 ? options.CommandTimeoutMs : 3000;
        var timeout = TimeSpan.FromMilliseconds(timeoutMs);

        if (!TryRun(binary, timeout, out _, out var startupError, "--version"))
        {
            return NatApplyResult.Fail($"iptables unavailable: {startupError}");
        }

        if (!EnsureChain(binary, timeout, NatTable, NatChain, out var chainError))
        {
            return NatApplyResult.Fail(chainError ?? "ensure nat chain failed");
        }

        if (!EnsureJump(binary, timeout, NatTable, "PREROUTING", NatChain, out var preroutingError))
        {
            return NatApplyResult.Fail(preroutingError ?? "ensure PREROUTING jump failed");
        }

        if (!EnsureJump(binary, timeout, NatTable, "OUTPUT", NatChain, out var outputError))
        {
            return NatApplyResult.Fail(outputError ?? "ensure OUTPUT jump failed");
        }

        if (!TryRun(binary, timeout, out _, out var flushError, "-t", NatTable, "-F", NatChain))
        {
            return NatApplyResult.Fail($"flush nat chain failed: {flushError}");
        }

        foreach (var rule in rules)
        {
            var args = new List<string>
            {
                "-t", NatTable,
                "-A", NatChain,
                "-p", "tcp"
            };

            if (!rule.ListenIp.Equals(IPAddress.Any) && !rule.ListenIp.Equals(IPAddress.IPv6Any))
            {
                args.Add("-d");
                args.Add(rule.ListenIp.ToString());
            }

            args.Add("--dport");
            args.Add(rule.ListenPort.ToString());
            args.Add("-j");
            args.Add("DNAT");
            args.Add("--to-destination");
            args.Add($"{rule.TargetIp}:{rule.TargetPort}");

            if (!TryRun(binary, timeout, out _, out var addError, args.ToArray()))
            {
                return NatApplyResult.Fail($"append nat rule failed for {rule.Key}: {addError}");
            }
        }

        lock (_stateLock)
        {
            _states = rules
                .Select(r => new StreamListenerState(
                    r.Key,
                    r.StreamId,
                    $"{r.ListenIp}:{r.ListenPort}",
                    true,
                    0,
                    null))
                .ToArray();
        }

        _logger.LogInformation("kernel nat applied rules={Count}", rules.Count);
        return NatApplyResult.Ok(rules.Count);
    }

    public bool Clear(StreamRuntimeOptions options, out string? error)
    {
        error = null;
        if (!OperatingSystem.IsLinux())
        {
            lock (_stateLock)
            {
                _states = Array.Empty<StreamListenerState>();
            }

            return true;
        }

        var binary = string.IsNullOrWhiteSpace(options.IptablesBinary) ? "iptables" : options.IptablesBinary.Trim();
        var timeoutMs = options.CommandTimeoutMs > 0 ? options.CommandTimeoutMs : 3000;
        var timeout = TimeSpan.FromMilliseconds(timeoutMs);

        _ = TryDeleteJump(binary, timeout, NatTable, "PREROUTING", NatChain);
        _ = TryDeleteJump(binary, timeout, NatTable, "OUTPUT", NatChain);
        _ = TryRun(binary, timeout, out _, out _, "-t", NatTable, "-F", NatChain);
        _ = TryRun(binary, timeout, out _, out _, "-t", NatTable, "-X", NatChain);

        lock (_stateLock)
        {
            _states = Array.Empty<StreamListenerState>();
        }

        return true;
    }

    public IReadOnlyCollection<StreamListenerState> GetStates()
    {
        lock (_stateLock)
        {
            return _states;
        }
    }

    private static List<NatRulePlan> BuildRules(IReadOnlyList<StreamListenerPlan> plans, List<string> errors)
    {
        var rules = new List<NatRulePlan>();
        foreach (var plan in plans)
        {
            var enabledTargets = plan.Targets
                .Where(t => t != null)
                .Where(t => t.Enable || !plan.Targets.Any(x => x.Enable))
                .ToList();

            if (enabledTargets.Count != 1)
            {
                errors.Add($"nat mode requires exactly one enabled target per stream: {plan.Key}");
                continue;
            }

            var target = enabledTargets[0];
            if (!StreamRouteCompiler.TryParseTarget(target.Addr ?? string.Empty, out var host, out var targetPort, out var parseError))
            {
                errors.Add($"nat target invalid for {plan.Key}: {parseError}");
                continue;
            }

            if (!IPAddress.TryParse(host, out var targetIp))
            {
                errors.Add($"nat target must be an ip address for {plan.Key}: {host}");
                continue;
            }

            rules.Add(new NatRulePlan(
                plan.Key,
                plan.StreamId,
                plan.ListenIp,
                plan.ListenPort,
                targetIp,
                targetPort));
        }

        return rules;
    }

    private static bool EnsureChain(string binary, TimeSpan timeout, string table, string chain, out string? error)
    {
        if (TryRun(binary, timeout, out _, out _, "-t", table, "-L", chain))
        {
            error = null;
            return true;
        }

        if (TryRun(binary, timeout, out _, out var createError, "-t", table, "-N", chain))
        {
            error = null;
            return true;
        }

        error = $"create chain failed: {createError}";
        return false;
    }

    private static bool EnsureJump(string binary, TimeSpan timeout, string table, string baseChain, string jumpChain, out string? error)
    {
        if (TryRun(binary, timeout, out _, out _, "-t", table, "-C", baseChain, "-j", jumpChain))
        {
            error = null;
            return true;
        }

        if (TryRun(binary, timeout, out _, out var addError, "-t", table, "-A", baseChain, "-j", jumpChain))
        {
            error = null;
            return true;
        }

        error = $"add jump {baseChain}->{jumpChain} failed: {addError}";
        return false;
    }

    private static bool TryDeleteJump(string binary, TimeSpan timeout, string table, string baseChain, string jumpChain)
    {
        if (!TryRun(binary, timeout, out _, out _, "-t", table, "-C", baseChain, "-j", jumpChain))
        {
            return true;
        }

        return TryRun(binary, timeout, out _, out _, "-t", table, "-D", baseChain, "-j", jumpChain);
    }

    private static bool TryRun(string binary, TimeSpan timeout, out string output, out string error, params string[] args)
    {
        output = string.Empty;
        error = string.Empty;
        try
        {
            var psi = new ProcessStartInfo
            {
                FileName = binary,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                CreateNoWindow = true
            };

            foreach (var arg in args)
            {
                psi.ArgumentList.Add(arg);
            }

            using var process = new Process { StartInfo = psi };
            process.Start();
            if (!process.WaitForExit((int)timeout.TotalMilliseconds))
            {
                try
                {
                    process.Kill(entireProcessTree: true);
                }
                catch
                {
                    // ignore kill errors
                }

                error = $"timeout after {(int)timeout.TotalMilliseconds}ms";
                return false;
            }

            output = process.StandardOutput.ReadToEnd().Trim();
            var stderr = process.StandardError.ReadToEnd().Trim();
            if (process.ExitCode == 0)
            {
                return true;
            }

            error = string.IsNullOrWhiteSpace(stderr)
                ? $"exit_code={process.ExitCode}"
                : stderr;
            return false;
        }
        catch (Exception ex)
        {
            error = ex.Message;
            return false;
        }
    }

    public sealed record NatApplyResult(bool Success, int RuleCount, IReadOnlyList<string> Errors)
    {
        public static NatApplyResult Ok(int ruleCount) => new(true, ruleCount, Array.Empty<string>());
        public static NatApplyResult Fail(string error) => new(false, 0, new[] { error });
        public static NatApplyResult Fail(IReadOnlyList<string> errors) => new(false, 0, errors);
    }

    private sealed record NatRulePlan(
        string Key,
        long StreamId,
        IPAddress ListenIp,
        int ListenPort,
        IPAddress TargetIp,
        int TargetPort);
}
