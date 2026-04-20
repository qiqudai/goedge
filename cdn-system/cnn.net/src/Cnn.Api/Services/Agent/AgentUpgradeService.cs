using System.Text.RegularExpressions;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Agent;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Agent;

public interface IAgentUpgradeService
{
    Task<ServiceResult<AgentUpgradeInfo>> GetUpgradeInfoAsync(long nodeId, CancellationToken cancellationToken);
}

public sealed class AgentUpgradeService : IAgentUpgradeService
{
    private static readonly Regex VersionRegex = new(@"v?(\d+(\.\d+)+)", RegexOptions.Compiled);

    private readonly INodeConfigService _nodeConfigService;
    private readonly ISystemConfigService _systemConfigService;

    public AgentUpgradeService(INodeConfigService nodeConfigService, ISystemConfigService systemConfigService)
    {
        _nodeConfigService = nodeConfigService;
        _systemConfigService = systemConfigService;
    }

    public async Task<ServiceResult<AgentUpgradeInfo>> GetUpgradeInfoAsync(long nodeId, CancellationToken cancellationToken)
    {
        if (nodeId <= 0)
        {
            return ServiceResult<AgentUpgradeInfo>.Fail(ErrorCodes.MissingParam);
        }

        var apiVersion = ReadAgentBinaryVersion();
        var nodeVersion = await _nodeConfigService.GetValueAsync(nodeId, "agent_version", cancellationToken);

        var system = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        var autoUpgrade = system.TryGetValue("auto_upgrade_agent", out var raw) &&
                          _systemConfigService.ParseBoolFlag(raw);

        var needUpgrade = CompareVersion(apiVersion, nodeVersion ?? string.Empty) > 0;
        var info = new AgentUpgradeInfo
        {
            ApiVersion = apiVersion,
            NodeVersion = nodeVersion,
            AutoUpgrade = autoUpgrade,
            NeedUpgrade = needUpgrade,
            ShouldUpgrade = needUpgrade && autoUpgrade
        };

        return ServiceResult<AgentUpgradeInfo>.Ok(info);
    }

    private static string ReadAgentBinaryVersion()
    {
        var env = Environment.GetEnvironmentVariable("AGENT_VERSION");
        if (!string.IsNullOrWhiteSpace(env))
        {
            return env.Trim();
        }

        var path = ResolveAgentBinaryPath();
        if (string.IsNullOrWhiteSpace(path))
        {
            return "unknown";
        }

        var name = Path.GetFileName(path);
        var extracted = ExtractVersionFromName(name);
        if (!string.IsNullOrWhiteSpace(extracted))
        {
            return extracted;
        }

        try
        {
            var info = new FileInfo(path);
            return info.LastWriteTime.ToString("yyyyMMdd-HHmmss");
        }
        catch
        {
            return "unknown";
        }
    }

    private static string ResolveAgentBinaryPath()
    {
        var baseDir = AppContext.BaseDirectory ?? string.Empty;
        var fileName = OperatingSystem.IsWindows() ? "cdn-agent.exe" : "cdn-agent";
        var candidate = Path.Combine(baseDir, "agent", fileName);
        if (File.Exists(candidate))
        {
            return candidate;
        }

        return string.Empty;
    }

    private static string ExtractVersionFromName(string? name)
    {
        if (string.IsNullOrWhiteSpace(name))
        {
            return string.Empty;
        }

        var match = VersionRegex.Match(name);
        if (!match.Success || match.Groups.Count < 2)
        {
            return string.Empty;
        }

        return match.Groups[1].Value;
    }

    private static int CompareVersion(string? a, string? b)
    {
        var left = ParseVersionSegments(a);
        var right = ParseVersionSegments(b);
        if (left.Count == 0 || right.Count == 0)
        {
            return 0;
        }

        var max = Math.Max(left.Count, right.Count);
        for (var i = 0; i < max; i++)
        {
            var lv = i < left.Count ? left[i] : 0;
            var rv = i < right.Count ? right[i] : 0;
            if (lv > rv)
            {
                return 1;
            }
            if (lv < rv)
            {
                return -1;
            }
        }

        return 0;
    }

    private static List<int> ParseVersionSegments(string? raw)
    {
        var value = raw?.Trim().TrimStart('v', 'V') ?? string.Empty;
        if (string.IsNullOrWhiteSpace(value))
        {
            return new List<int>();
        }

        var parts = value.Split('.', StringSplitOptions.RemoveEmptyEntries);
        var list = new List<int>(parts.Length);
        foreach (var part in parts)
        {
            var digits = LeadingDigits(part.Trim());
            if (string.IsNullOrWhiteSpace(digits))
            {
                break;
            }

            if (!int.TryParse(digits, out var number))
            {
                return new List<int>();
            }

            list.Add(number);
        }

        return list;
    }

    private static string LeadingDigits(string raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return string.Empty;
        }

        for (var i = 0; i < raw.Length; i++)
        {
            var ch = raw[i];
            if (ch < '0' || ch > '9')
            {
                return raw.Substring(0, i);
            }
        }

        return raw;
    }
}
