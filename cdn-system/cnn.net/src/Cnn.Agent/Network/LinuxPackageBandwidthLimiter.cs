using System.Diagnostics;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Network;

public sealed class LinuxPackageBandwidthLimiter : IPackageBandwidthLimiter
{
    private readonly ILogger<LinuxPackageBandwidthLimiter> _logger;
    private const long MinLimitMbps = 1;
    private const int DefaultTimeoutMs = 5000;

    public LinuxPackageBandwidthLimiter(ILogger<LinuxPackageBandwidthLimiter> logger)
    {
        _logger = logger;
    }

    public Task<PackageBandwidthApplyResult> ApplyAsync(
        IReadOnlyCollection<AgentPackageConfigDto> packages,
        CancellationToken cancellationToken)
    {
        if (!OperatingSystem.IsLinux())
        {
            return Task.FromResult(new PackageBandwidthApplyResult(false, string.Empty, 0, "non-linux platform skipped"));
        }

        var iface = ResolveInterface();
        if (string.IsNullOrWhiteSpace(iface))
        {
            return Task.FromResult(new PackageBandwidthApplyResult(false, string.Empty, 0, "default network interface not found"));
        }

        var requested = ResolveRequestedLimitMbps(packages);
        var limit = requested <= 0 ? MinLimitMbps : requested;
        if (requested <= 0)
        {
            _logger.LogWarning(
                "package bandwidth is non-positive ({Requested}), forcing minimum limit {Limit}Mbps",
                requested,
                limit);
        }

        if (!TryRun("tc", out var err, "qdisc", "replace", "dev", iface, "root", "tbf", "rate", $"{limit}mbit", "burst", "256kb", "latency", "50ms"))
        {
            return Task.FromResult(new PackageBandwidthApplyResult(false, iface, limit, $"egress limit apply failed: {err}"));
        }

        if (!TryRun("tc", out err, "qdisc", "replace", "dev", iface, "handle", "ffff:", "ingress"))
        {
            return Task.FromResult(new PackageBandwidthApplyResult(false, iface, limit, $"ingress qdisc apply failed: {err}"));
        }

        if (!TryRun("tc", out err,
                "filter", "replace", "dev", iface, "parent", "ffff:",
                "protocol", "all", "prio", "1",
                "u32", "match", "u32", "0", "0",
                "police", "rate", $"{limit}mbit", "burst", "256kb", "mtu", "64kb", "drop", "flowid", ":1"))
        {
            return Task.FromResult(new PackageBandwidthApplyResult(false, iface, limit, $"ingress limit apply failed: {err}"));
        }

        return Task.FromResult(new PackageBandwidthApplyResult(true, iface, limit, "ok"));
    }

    private static long ResolveRequestedLimitMbps(IReadOnlyCollection<AgentPackageConfigDto> packages)
    {
        long max = 0;
        foreach (var pkg in packages)
        {
            if (!string.Equals(pkg.Status, "active", StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            var value = ParseBandwidthMbps(pkg.Limits?.Bandwidth);
            if (value > max)
            {
                max = value;
            }
        }

        return max;
    }

    private static long ParseBandwidthMbps(string? raw)
    {
        var value = raw?.Trim().ToLowerInvariant() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(value))
        {
            return 0;
        }

        var multiplier = 1d;
        if (value.EndsWith("g", StringComparison.Ordinal))
        {
            multiplier = 1024d;
            value = value[..^1];
        }
        else if (value.EndsWith("m", StringComparison.Ordinal))
        {
            value = value[..^1];
        }
        else if (value.EndsWith("k", StringComparison.Ordinal))
        {
            multiplier = 1d / 1024d;
            value = value[..^1];
        }

        if (!double.TryParse(value.Trim(), out var parsed))
        {
            return 0;
        }

        var result = (long)Math.Floor(parsed * multiplier);
        return result > 0 ? result : 0;
    }

    private static string ResolveInterface()
    {
        if (!TryRun("ip", out _, out var output, "route", "show", "default"))
        {
            return string.Empty;
        }

        var lines = output.Split('\n', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
        foreach (var line in lines)
        {
            var parts = line.Split(' ', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
            for (var i = 0; i < parts.Length - 1; i++)
            {
                if (!string.Equals(parts[i], "dev", StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }

                return parts[i + 1];
            }
        }

        return string.Empty;
    }

    private static bool TryRun(string fileName, out string error, params string[] args)
        => TryRun(fileName, out error, out _, args);

    private static bool TryRun(string fileName, out string error, out string output, params string[] args)
    {
        error = string.Empty;
        output = string.Empty;
        try
        {
            var psi = new ProcessStartInfo
            {
                FileName = fileName,
                RedirectStandardError = true,
                RedirectStandardOutput = true,
                UseShellExecute = false,
                CreateNoWindow = true
            };
            foreach (var arg in args)
            {
                psi.ArgumentList.Add(arg);
            }

            using var process = new Process { StartInfo = psi };
            process.Start();
            if (!process.WaitForExit(DefaultTimeoutMs))
            {
                try
                {
                    process.Kill(entireProcessTree: true);
                }
                catch
                {
                }

                error = $"timeout after {DefaultTimeoutMs}ms";
                return false;
            }

            output = process.StandardOutput.ReadToEnd().Trim();
            var stderr = process.StandardError.ReadToEnd().Trim();
            if (process.ExitCode == 0)
            {
                return true;
            }

            error = string.IsNullOrWhiteSpace(stderr) ? $"exit_code={process.ExitCode}" : stderr;
            return false;
        }
        catch (Exception ex)
        {
            error = ex.Message;
            return false;
        }
    }
}

