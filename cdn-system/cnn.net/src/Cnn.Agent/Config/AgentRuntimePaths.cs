using Microsoft.Extensions.Configuration;

namespace Cnn.Agent.Config;

public sealed class AgentRuntimePaths
{
    public string WorkDir { get; }
    public string RuntimeRoot { get; }
    public string ConfDir => Path.Combine(RuntimeRoot, "conf");
    public string CacheDir => Path.Combine(RuntimeRoot, "cache");
    public string PackagesDir => Path.Combine(RuntimeRoot, "packages");
    public string CertDir => Path.Combine(RuntimeRoot, "cert");
    public string LogsDir => Path.Combine(RuntimeRoot, "logs");
    public string PluginsDir => Path.Combine(RuntimeRoot, "plugins");
    public string ConfigPath => Path.Combine(ConfDir, "cdn_config.json");
    public string ConfigBackupPath => ConfigPath + ".bak";
    public string ResourcesPath => Path.Combine(ConfDir, "resources.json");
    public string ErrorPagesPath => Path.Combine(ConfDir, "error_pages.json");
    public string DefaultConfigPath => Path.Combine(ConfDir, "default_config.json");
    public string CcRulesPath => Path.Combine(ConfDir, "cc_rules.json");
    public string CcMatchersPath => Path.Combine(ConfDir, "cc_matchers.json");
    public string CcFiltersPath => Path.Combine(ConfDir, "cc_filters.json");
    public string L2StatusPath => Path.Combine(ConfDir, "l2_status.json");
    public string DebugSwitchPath => Path.Combine(ConfDir, "debug_switches.json");
    public string PluginStatePath => Path.Combine(ConfDir, "plugin_state.json");
    public string SyncStatePath => Path.Combine(ConfDir, "sync_state.json");
    public string TaskIdempotencyPath => Path.Combine(ConfDir, "task_idempotency.json");
    public string TaskAckOutboxPath => Path.Combine(ConfDir, "task_ack_outbox.json");
    public string ManualDebugLogPath => Path.Combine(LogsDir, "manual_debug.jsonl");

    public AgentRuntimePaths(IConfiguration configuration)
    {
        WorkDir = ResolveWorkDir();
        var configuredRuntimeRoot = configuration["Agent:RuntimeRoot"] ?? configuration["Runtime:Root"];
        if (string.IsNullOrWhiteSpace(configuredRuntimeRoot))
        {
            RuntimeRoot = Path.Combine(WorkDir, "edge-node");
            return;
        }

        var value = configuredRuntimeRoot.Trim();
        RuntimeRoot = Path.IsPathRooted(value)
            ? value
            : Path.Combine(WorkDir, value);
    }

    public static string ResolveWorkDir()
    {
        var baseDir = AppContext.BaseDirectory;
        if (string.IsNullOrWhiteSpace(baseDir))
        {
            var processPath = Environment.ProcessPath;
            if (!string.IsNullOrWhiteSpace(processPath))
            {
                baseDir = Path.GetDirectoryName(processPath) ?? string.Empty;
            }
        }

        return baseDir.TrimEnd(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar);
    }

    public static string ResolveCacheRoot(string? configuredRoot, string runtimeRoot)
    {
        if (string.IsNullOrWhiteSpace(configuredRoot) ||
            string.Equals(configuredRoot.Trim(), "www", StringComparison.OrdinalIgnoreCase))
        {
            return Path.Combine(runtimeRoot, "cache");
        }

        var trimmed = NormalizeRelativePath(configuredRoot);
        if (string.IsNullOrWhiteSpace(trimmed))
        {
            return Path.Combine(runtimeRoot, "cache");
        }

        return Path.Combine(runtimeRoot, trimmed);
    }

    private static string NormalizeRelativePath(string path)
    {
        var trimmed = path.Trim();
        if (string.IsNullOrWhiteSpace(trimmed))
        {
            return string.Empty;
        }

        if (!Path.IsPathRooted(trimmed))
        {
            return trimmed;
        }

        var root = Path.GetPathRoot(trimmed);
        if (string.IsNullOrWhiteSpace(root))
        {
            return trimmed.TrimStart(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar);
        }

        var withoutRoot = trimmed.Substring(root.Length);
        return withoutRoot.TrimStart(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar);
    }
}
