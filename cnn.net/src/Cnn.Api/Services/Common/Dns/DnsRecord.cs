namespace Cnn.Api.Services.Common.Dns;

public sealed class DnsRecord
{
    public string Type { get; set; } = string.Empty;
    public string Name { get; set; } = string.Empty;
    public string Value { get; set; } = string.Empty;
    public string Line { get; set; } = string.Empty;
    public int TTL { get; set; } = 600;
    public int Weight { get; set; }
}
