using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class StatRankingItemDto
{
    [JsonPropertyName("rank")]
    public int Rank { get; set; }

    [JsonPropertyName("item")]
    public string? Item { get; set; }

    [JsonPropertyName("request_count")]
    public int RequestCount { get; set; }

    [JsonPropertyName("out_traffic")]
    public string? OutTraffic { get; set; }

    [JsonPropertyName("origin_traffic")]
    public string? OriginTraffic { get; set; }
}

public sealed class StatLatencyItemDto
{
    [JsonPropertyName("rank")]
    public int Rank { get; set; }

    [JsonPropertyName("item")]
    public string? Item { get; set; }

    [JsonPropertyName("request_count")]
    public int RequestCount { get; set; }

    [JsonPropertyName("avg_time")]
    public double AvgTime { get; set; }

    [JsonPropertyName("max_time")]
    public double MaxTime { get; set; }

    [JsonPropertyName("min_time")]
    public double MinTime { get; set; }

    [JsonPropertyName("p95_time")]
    public double P95Time { get; set; }
}

public sealed class StatRankingResultDto
{
    [JsonPropertyName("list")]
    public IReadOnlyList<StatRankingItemDto> List { get; set; } = Array.Empty<StatRankingItemDto>();
}

public sealed class StatLatencyResultDto
{
    [JsonPropertyName("list")]
    public IReadOnlyList<StatLatencyItemDto> List { get; set; } = Array.Empty<StatLatencyItemDto>();
}

public sealed class StatBasicResultDto
{
    [JsonPropertyName("x_axis")]
    public IReadOnlyList<string> XAxis { get; set; } = Array.Empty<string>();

    [JsonPropertyName("bandwidth")]
    public IReadOnlyList<double> Bandwidth { get; set; } = Array.Empty<double>();

    [JsonPropertyName("traffic")]
    public IReadOnlyList<double> Traffic { get; set; } = Array.Empty<double>();

    [JsonPropertyName("qps")]
    public IReadOnlyList<double> Qps { get; set; } = Array.Empty<double>();
}

public sealed class StatQualityResultDto
{
    [JsonPropertyName("x_axis")]
    public IReadOnlyList<string> XAxis { get; set; } = Array.Empty<string>();

    [JsonPropertyName("hit_rate")]
    public IReadOnlyList<double> HitRate { get; set; } = Array.Empty<double>();

    [JsonPropertyName("status_4xx")]
    public IReadOnlyList<double> Status4xx { get; set; } = Array.Empty<double>();

    [JsonPropertyName("status_5xx")]
    public IReadOnlyList<double> Status5xx { get; set; } = Array.Empty<double>();
}

public sealed class StatOriginResultDto
{
    [JsonPropertyName("x_axis")]
    public IReadOnlyList<string> XAxis { get; set; } = Array.Empty<string>();

    [JsonPropertyName("origin_bandwidth")]
    public IReadOnlyList<double> OriginBandwidth { get; set; } = Array.Empty<double>();

    [JsonPropertyName("origin_traffic")]
    public IReadOnlyList<double> OriginTraffic { get; set; } = Array.Empty<double>();
}

public sealed class StatNodeTrafficDto
{
    [JsonPropertyName("x_axis")]
    public IReadOnlyList<string> XAxis { get; set; } = Array.Empty<string>();

    [JsonPropertyName("in_traffic")]
    public IReadOnlyList<double> InTraffic { get; set; } = Array.Empty<double>();

    [JsonPropertyName("out_traffic")]
    public IReadOnlyList<double> OutTraffic { get; set; } = Array.Empty<double>();
}

public sealed class StatNodeRankingItemDto
{
    [JsonPropertyName("rank")]
    public int Rank { get; set; }

    [JsonPropertyName("node")]
    public string? Node { get; set; }

    [JsonPropertyName("nic")]
    public string? Nic { get; set; }

    [JsonPropertyName("out")]
    public string? Out { get; set; }

    [JsonPropertyName("in")]
    public string? In { get; set; }
}

public sealed class StatNodeRankingResultDto
{
    [JsonPropertyName("list")]
    public IReadOnlyList<StatNodeRankingItemDto> List { get; set; } = Array.Empty<StatNodeRankingItemDto>();
}

public sealed class StatNodeMetricPointDto
{
    [JsonPropertyName("time")]
    public string? Time { get; set; }

    [JsonPropertyName("value")]
    public double Value { get; set; }
}

public sealed class StatNodeMetricsResultDto
{
    [JsonPropertyName("list")]
    public IReadOnlyList<StatNodeMetricPointDto> List { get; set; } = Array.Empty<StatNodeMetricPointDto>();
}
