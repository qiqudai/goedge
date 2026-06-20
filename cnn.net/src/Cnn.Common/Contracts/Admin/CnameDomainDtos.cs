using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed record CnameDomainListResult(IReadOnlyList<CnameDomainItem> List);

public sealed class CnameDomainItem
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("domain")]
    public string Domain { get; set; } = string.Empty;

    [JsonPropertyName("dns_provider_id")]
    public long DnsProviderId { get; set; }

    [JsonPropertyName("note")]
    public string? Note { get; set; }

    [JsonPropertyName("created_at")]
    public DateTime? CreatedAt { get; set; }

    [JsonPropertyName("updated_at")]
    public DateTime? UpdatedAt { get; set; }
}

public sealed class CnameDomainUpsertRequest
{
    [JsonPropertyName("domain")]
    public string? Domain { get; set; }

    [JsonPropertyName("note")]
    public string? Note { get; set; }

    [JsonPropertyName("dns_provider_id")]
    public long? DnsProviderId { get; set; }
}
