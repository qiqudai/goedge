using System.Text.Json.Serialization;

namespace Cnn.Agent.Plugin;

public sealed class PluginManifest
{
    [JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [JsonPropertyName("version")]
    public string Version { get; set; } = string.Empty;

    [JsonPropertyName("sha256")]
    public string Sha256 { get; set; } = string.Empty;

    [JsonPropertyName("entry_type")]
    public string EntryType { get; set; } = string.Empty;

    [JsonPropertyName("entry_assembly")]
    public string? EntryAssembly { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("signature")]
    public string? Signature { get; set; }

    [JsonPropertyName("settings")]
    public Dictionary<string, string>? Settings { get; set; }
}

public sealed class PluginRuntimeOptions
{
    public bool Enabled { get; set; } = false;
    public string Directory { get; set; } = "plugins";
    public bool RequireSignature { get; set; } = true;
    public int MaxAssemblyBytes { get; set; } = 32 * 1024 * 1024;
    public bool RestrictAssemblyToPluginDirectory { get; set; } = true;
    public int EvalTimeoutMs { get; set; } = 5;
    public int ScanIntervalSeconds { get; set; } = 3;
    public string? SignaturePublicKeyPath { get; set; }
    public string? SignatureHmacKey { get; set; }
    public HashSet<string> AllowedPluginNames { get; set; } = new(StringComparer.OrdinalIgnoreCase);
    public PluginBreakerOptions Breaker { get; set; } = new();
}
