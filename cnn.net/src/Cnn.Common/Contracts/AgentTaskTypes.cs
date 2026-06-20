namespace Cnn.Common.Contracts;

public static class AgentTaskTypes
{
    public const string IssueCert = "issue_cert";
    public const string DeployCert = "deploy_cert";
    public const string RefreshUrl = "refresh_url";
    public const string RefreshDir = "refresh_dir";
    public const string ClearCache = "clear_cache";
    public const string Preheat = "preheat";
    public const string ConfigSync = "config_sync";
    public const string AgentUpgrade = "agent_upgrade";
    public const string DebugSwitch = "debug_switch";
    public const string DebugLogSwitch = "debug_log_switch";
    public const string ManualDebugLog = "manual_debug_log";
    public const string DebugLogWrite = "debug_log_write";

    private static readonly HashSet<string> Supported = new(StringComparer.OrdinalIgnoreCase)
    {
        IssueCert,
        DeployCert,
        RefreshUrl,
        RefreshDir,
        ClearCache,
        Preheat,
        ConfigSync,
        AgentUpgrade,
        DebugSwitch,
        DebugLogSwitch,
        ManualDebugLog,
        DebugLogWrite,
        "sync_package",
        "package_sync"
    };

    public static bool IsSupported(string? taskType)
    {
        return !string.IsNullOrWhiteSpace(taskType) && Supported.Contains(taskType.Trim());
    }

    public static string Normalize(string taskType)
    {
        return taskType.Trim().ToLowerInvariant();
    }
}
