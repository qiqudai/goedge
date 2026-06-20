using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Agent;

public sealed class AgentIssuedCertRequest
{
    [JsonPropertyName("cert_id")]
    public long CertId { get; set; }

    [JsonPropertyName("cert")]
    public string? CertPem { get; set; }

    [JsonPropertyName("key")]
    public string? KeyPem { get; set; }

    [JsonPropertyName("issue_task_id")]
    public long IssueTaskId { get; set; }

    [JsonPropertyName("rate_limited")]
    public bool RateLimited { get; set; }

    [JsonPropertyName("rate_cooldown")]
    public int RateCooldown { get; set; }
}

public sealed class IssueCertTaskPayload
{
    [JsonPropertyName("ca")]
    public string? Ca { get; set; }

    [JsonPropertyName("ca_dir_url")]
    public string? CaDirUrl { get; set; }

    [JsonPropertyName("email")]
    public string? Email { get; set; }

    [JsonPropertyName("items")]
    public List<IssueCertItem> Items { get; set; } = new();
}

public sealed class IssueCertItem
{
    [JsonPropertyName("cert_id")]
    public long CertId { get; set; }

    [JsonPropertyName("domains")]
    public IReadOnlyList<string> Domains { get; set; } = Array.Empty<string>();
}

public sealed class DeployCertTaskPayload
{
    [JsonPropertyName("cert_id")]
    public long CertId { get; set; }

    [JsonPropertyName("cert")]
    public string? CertPem { get; set; }

    [JsonPropertyName("key")]
    public string? KeyPem { get; set; }

    [JsonPropertyName("domains")]
    public IReadOnlyList<string> Domains { get; set; } = Array.Empty<string>();
}

