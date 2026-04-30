using System.Diagnostics;
using System.Globalization;

namespace Cnn.Agent.Network;

public sealed class LinuxNodeBandwidthLimiter : INodeBandwidthLimiter
{
    private const int DefaultTimeoutMs = 5000;

    public Task<NodeBandwidthApplyResult> ApplyAsync(string? bwLimit, CancellationToken cancellationToken)
    {
        if (!OperatingSystem.IsLinux())
        {
            return Task.FromResult(new NodeBandwidthApplyResult(false, string.Empty, 0, "non-linux platform skipped"));
        }

        var iface = ResolveInterface();
        if (string.IsNullOrWhiteSpace(iface))
        {
            return Task.FromResult(new NodeBandwidthApplyResult(false, string.Empty, 0, "default network interface not found"));
        }

        var limit = ParseBandwidthMbps(bwLimit);
        if (limit <= 0)
        {
            TryRun("tc", out _, "qdisc", "del", "dev", iface, "root");
            TryRun("tc", out _, "qdisc", "del", "dev", iface, "ingress");
            return Task.FromResult(new NodeBandwidthApplyResult(true, iface, 0, "unlimited"));
        }

        if (!TryRun("tc", out var err, "qdisc", "replace", "dev", iface, "root", "tbf", "rate", $"{limit}mbit", "burst", "256kb", "latency", "50ms"))
        {
            return Task.FromResult(new NodeBandwidthApplyResult(false, iface, limit, $"egress limit apply failed: {err}"));
        }

        if (!TryRun("tc", out err, "qdisc", "replace", "dev", iface, "handle", "ffff:", "ingress"))
        {
            return Task.FromResult(new NodeBandwidthApplyResult(false, iface, limit, $"ingress qdisc apply failed: {err}"));
        }

        if (!TryRun("tc", out err,
                "filter", "replace", "dev", iface, "parent", "ffff:",
                "protocol", "all", "prio", "1",
                "u32", "match", "u32", "0", "0",
                "police", "rate", $"{limit}mbit", "burst", "256kb", "mtu", "64kb", "drop", "flowid", ":1"))
        {
            return Task.FromResult(new NodeBandwidthApplyResult(false, iface, limit, $"ingress limit apply failed: {err}"));
        }

        return Task.FromResult(new NodeBandwidthApplyResult(true, iface, limit, "ok"));
    }

    private static long ParseBandwidthMbps(string? raw)
    {
        var value = raw?.Trim().ToLowerInvariant() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(value) || value is "0" or "unlimited" or "unlimit")
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

        if (!double.TryParse(value.Trim(), NumberStyles.Float, CultureInfo.InvariantCulture, out var parsed))
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
                if (string.Equals(parts[i], "dev", StringComparison.OrdinalIgnoreCase))
                {
                    return parts[i + 1];
                }
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

