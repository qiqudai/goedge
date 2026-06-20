using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class UsagePointDto
{
    [JsonPropertyName("time")]
    public string? Time { get; set; }

    [JsonPropertyName("value")]
    public double Value { get; set; }
}

public sealed class UsageResultDto
{
    [JsonPropertyName("x_axis")]
    public IReadOnlyList<string> XAxis { get; set; } = Array.Empty<string>();

    [JsonPropertyName("values")]
    public IReadOnlyList<double> Values { get; set; } = Array.Empty<double>();

    [JsonPropertyName("list")]
    public IReadOnlyList<UsagePointDto> List { get; set; } = Array.Empty<UsagePointDto>();

    [JsonPropertyName("total")]
    public double Total { get; set; }

    [JsonPropertyName("avg")]
    public double Avg { get; set; }

    [JsonPropertyName("peak")]
    public double Peak { get; set; }

    [JsonPropertyName("unit")]
    public string? Unit { get; set; }
}

public sealed class DomainUsageDto
{
    [JsonPropertyName("total_domains")]
    public int TotalDomains { get; set; }

    [JsonPropertyName("total_main_domains")]
    public int TotalMainDomains { get; set; }

    [JsonPropertyName("domain_limit")]
    public int DomainLimit { get; set; }

    [JsonPropertyName("main_domain_limit")]
    public int MainDomainLimit { get; set; }

    [JsonPropertyName("exceeded")]
    public bool Exceeded { get; set; }

    [JsonPropertyName("message")]
    public string? Message { get; set; }
}
